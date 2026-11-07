import {ref, computed, watch} from 'vue'
import {
  AIGenerateDDLStream,
  AIPartitionMicroservices,
  AIValidateDDL,
  AICancelTask,
  AIPlanBackendCode,
  SetGenerateTests,
  AIApplyBackendCode,
  AIDiscardBackendPlan,
  AIReadBackendPlanFile,
  AIFindOpenAPIFiles,
  AIReviewScan,
  AIReviewProject,
  GetAIConfig,
  LoadAIArtifacts,
  SaveAIArtifact,
  LoadAIReviewReport,
} from '../../wailsjs/go/main/App'
import type {ai} from '../../wailsjs/go/models'
import {EventsOn} from '../../wailsjs/runtime'
import {useProject} from './project'

// ==================== 模式 ====================
// panel: 能力面板(每个能力独立触发);wizard: 新手向导(需求→DDL→划分→后端的固定链)。
// 两种形态共享同一份产物真值,产物以项目 .gowind/ai/ 目录为持久层:
// 后端在生成成功时自动落盘,这里负责恢复现场与显式保存手动编辑。
export type AIMode = 'panel' | 'wizard'

const mode = ref<AIMode>('panel')

// ==================== 产物状态 ====================
const requirements = ref('')
const ddlContent = ref('')
const ddlGenerating = ref(false)
// 当前流式尝试序号:>1 表示 DDL 未通过解析校验后的自动修复重试
const ddlAttempt = ref(1)

const partitions = ref<ai.MicroservicePartition[]>([])
const partitionWarnings = ref<string[]>([])
const partitionLoading = ref(false)
// 划分流式原始 JSON 预览(修复重试时 attempt 变化清空,与 DDL 同语义)
const partitionStreamContent = ref('')
let partitionStreamAttempt = 0
// 划分编辑态:ddlTables 是 DDL 全表清单(未分配池基准),进入编辑时快照
// partitions,保存或放弃时二选一,避免半途编辑污染生成链的真值。
const editingPartitions = ref(false)
const ddlTables = ref<string[]>([])
let partitionsSnapshot: ai.MicroservicePartition[] | null = null

const ormType = ref<'ent' | 'gorm'>('ent')
// 后端生成时是否附带 CRUD service 单测骨架(fake 仓储);plan 前同步给 Go 侧
const backendWithTests = ref(false)
async function setBackendWithTests(enable: boolean) {
  backendWithTests.value = enable
  await SetGenerateTests(enable)
}
// 后端生成走预览确认两阶段:预览树在服务端临时目录,确认后原样拷贝进项目
const backendPlan = ref<ai.BackendPlanResult | null>(null)
const planningLoading = ref(false)
const applyingLoading = ref(false)
// drawer 单文件内容预览
const previewFile = ref('')
const previewContent = ref('')
const previewDrawerOpen = ref(false)

const openapiFiles = ref<string[]>([])
const openapiMessage = ref('')
const openapiLoading = ref(false)

const reviewScanFiles = ref<ai.CodeFileInfo[]>([])
const reviewScanWarnings = ref<string[]>([])
const reviewScanLoading = ref(false)
const reviewSelected = ref<string[]>([])
const reviewLoading = ref(false)
const reviewResult = ref('')
const reviewReports = ref<ai.ReviewFile[]>([])
// 正在回看的历史报告名;空串表示展示实时/最新审查结果
const reviewViewName = ref('')
// map-reduce 批次进度与结构化 issue(ai:review-batch 事件驱动)
const reviewBatches = ref<{index: number; total: number; label: string; issueCount: number}[]>([])
const reviewBatchTotal = ref(0)
const reviewIssues = ref<ai.ReviewIssue[]>([])

const hasPartitions = computed(() => partitions.value.length > 0)

// AI 服务是否已配置 API Key(null=尚未检测):首次运行横幅与配置卡徽标共用
const aiConfigured = ref<boolean | null>(null)

async function checkConfigured() {
  try {
    const cfg = await GetAIConfig()
    aiConfigured.value = !!cfg && !!cfg.apiKey
  } catch {
    aiConfigured.value = false
  }
}

// ==================== 流式事件 ====================
// 后端在 DDL 校验失败自动重试时递增 attempt;前端见到变化必须清空
// 已累积内容,否则两次尝试的文本会拼在一起。
let ddlStreamAttempt = 0

function onAIStream(data: any) {
  if (!data || typeof data.delta !== 'string') return
  if (data.task === 'ddl' && ddlGenerating.value) {
    const attempt = typeof data.attempt === 'number' ? data.attempt : 1
    if (attempt !== ddlStreamAttempt) {
      ddlStreamAttempt = attempt
      ddlAttempt.value = attempt
      ddlContent.value = ''
    }
    ddlContent.value += data.delta
  } else if (data.task === 'partition' && partitionLoading.value) {
    const attempt = typeof data.attempt === 'number' ? data.attempt : 1
    if (attempt !== partitionStreamAttempt) {
      partitionStreamAttempt = attempt
      partitionStreamContent.value = ''
    }
    partitionStreamContent.value += data.delta
  } else if (data.task === 'review' && reviewLoading.value) {
    reviewResult.value += data.delta
  }
}

let eventsBound = false
function bindEvents() {
  if (eventsBound) return
  eventsBound = true
  // 绑定在 store 级单例上且不随页面卸载退订:生成/审查在后台继续,
  // 用户切走再切回仍能接上流式内容。
  EventsOn('ai:stream', onAIStream)
  // map-reduce 审查的批次进度:逐批点亮
  EventsOn('ai:review-batch', (data: any) => {
    if (!data || typeof data.index !== 'number') return
    reviewBatchTotal.value = data.total || 0
    reviewBatches.value = [
        ...reviewBatches.value.filter(b => b.index !== data.index),
        {
          index: data.index,
          total: data.total,
          label: typeof data.label === 'string' ? data.label : '',
          issueCount: typeof data.issueCount === 'number' ? data.issueCount : 0,
        },
    ].sort((a, b) => a.index - b.index)
  })
  // 审查报告落盘后只刷新报告列表——不能全量 loadArtifacts,
  // 否则磁盘值会覆盖用户尚未保存的 DDL/需求手动编辑。
  EventsOn('ai-review-saved', () => {
    refreshReports()
  })
}

// ==================== 产物恢复/重置 ====================
async function loadArtifacts() {
  try {
    const arts = await LoadAIArtifacts()
    requirements.value = arts.requirements || ''
    ddlContent.value = arts.ddl || ''
    partitions.value = parsePartitionsJSON(arts.partitions)
    reviewReports.value = arts.reviews || []
  } catch (e) {
    console.error('恢复 AI 产物失败:', e)
  }
}

// 只刷新历史报告列表(审查落盘后由 ai-review-saved 事件触发)
async function refreshReports() {
  try {
    const arts = await LoadAIArtifacts()
    reviewReports.value = arts.reviews || []
  } catch (e) {
    console.error('刷新审查报告列表失败:', e)
  }
}

function parsePartitionsJSON(raw: string): ai.MicroservicePartition[] {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

// 取消在途任务("ddl"/"partition"/"review"):在途调用自己的返回路径
// 会以「已取消」失败收场,现有 message.error 与 finally 复位逻辑自动收尾;
// DDL 已流出的部分内容保留。任务没在跑时返回 false,静默忽略即可。
async function cancelTask(task: 'ddl' | 'partition' | 'review'): Promise<boolean> {
  try {
    return await AICancelTask(task)
  } catch (e) {
    console.error('取消任务失败:', e)
    return false
  }
}

function resetForProject() {
  requirements.value = ''
  ddlContent.value = ''
  ddlGenerating.value = false
  ddlStreamAttempt = 0
  ddlAttempt.value = 1
  partitions.value = []
  partitionWarnings.value = []
  partitionLoading.value = false
  partitionStreamContent.value = ''
  partitionStreamAttempt = 0
  editingPartitions.value = false
  ddlTables.value = []
  partitionsSnapshot = null
  openapiFiles.value = []
  openapiMessage.value = ''
  openapiLoading.value = false
  reviewScanFiles.value = []
  reviewScanWarnings.value = []
  reviewSelected.value = []
  reviewLoading.value = false
  reviewResult.value = ''
  reviewViewName.value = ''
  reviewReports.value = []
  reviewBatches.value = []
  reviewBatchTotal.value = 0
  reviewIssues.value = []
  // 未确认的后端生成计划随项目作废(服务端临时目录一并清理)
  if (backendPlan.value) {
    AIDiscardBackendPlan(backendPlan.value.planId || '').catch(() => {})
  }
  backendPlan.value = null
  planningLoading.value = false
  applyingLoading.value = false
  previewFile.value = ''
  previewContent.value = ''
  previewDrawerOpen.value = false
}

// ==================== 各能力 action ====================
// 约定:action 返回后端原始结果对象,消息提示由组件用 t() 呈现。

async function generateDDL() {
  if (!requirements.value.trim() || ddlGenerating.value) return null
  ddlGenerating.value = true
  ddlContent.value = ''
  ddlStreamAttempt = 1
  ddlAttempt.value = 1
  try {
    const result = await AIGenerateDDLStream(requirements.value)
    // 返回值是权威完整内容(流里可能混有重试前的残段,以返回值收口)
    if (result.success) {
      ddlContent.value = result.content
    }
    return result
  } finally {
    ddlGenerating.value = false
  }
}

// 手动编辑后的显式保存:需求与 DDL 一起存(与生成时的自动落盘同目录同名)。
async function saveArtifacts(): Promise<string> {
  const [req, ddl] = await Promise.all([
    SaveAIArtifact('requirements', requirements.value),
    SaveAIArtifact('ddl', ddlContent.value),
  ])
  return ddl.success && req.success ? '' : (ddl.error || req.error || '保存失败')
}

async function partition() {
  if (!ddlContent.value.trim() || partitionLoading.value) return null
  partitionLoading.value = true
  partitionStreamContent.value = ''
  partitionStreamAttempt = 0
  try {
    const result = await AIPartitionMicroservices(ddlContent.value)
    if (result.success) {
      partitions.value = result.partitions || []
      partitionWarnings.value = result.warnings || []
      ddlTables.value = result.availableTables || []
    }
    return result
  } finally {
    partitionLoading.value = false
    partitionStreamContent.value = ''
  }
}

// ==================== 划分编辑 ====================
const unassignedTables = computed(() => {
  const assigned = new Set<string>()
  for (const p of partitions.value) {
    for (const t of p.tables) assigned.add(t)
  }
  return ddlTables.value.filter(t => !assigned.has(t))
})

// 进入编辑:快照当前划分 + 拉 DDL 表清单作未分配池基准
async function beginEditPartitions(): Promise<string> {
  if (editingPartitions.value) return ''
  const result = await AIValidateDDL(ddlContent.value)
  if (!result.success) return result.error || 'DDL 校验失败'
  ddlTables.value = result.tables || []
  partitionsSnapshot = JSON.parse(JSON.stringify(partitions.value))
  editingPartitions.value = true
  return ''
}

function cancelEditPartitions() {
  if (partitionsSnapshot) {
    partitions.value = partitionsSnapshot
    partitionsSnapshot = null
  }
  editingPartitions.value = false
}

// 保存编辑:空名/重名校验后持久化 partitions.json,退出编辑态。
// 返回空串表示成功,非空为错误信息(与后端错误串约定一致)。
async function savePartitions(): Promise<string> {
  const names = new Set<string>()
  for (const p of partitions.value) {
    const name = p.serviceName.trim()
    if (!name) return '服务名不能为空'
    if (names.has(name)) return `服务名重复: ${name}`
    names.add(name)
    p.serviceName = name
  }
  const result = await SaveAIArtifact('partitions', JSON.stringify(partitions.value, null, 2))
  if (!result.success) return result.error || '保存失败'
  partitionsSnapshot = null
  editingPartitions.value = false
  return ''
}

function removeTable(idx: number, table: string) {
  const p = partitions.value[idx]
  if (!p) return
  p.tables = p.tables.filter(t => t !== table)
}

// 表在服务间移动:目标与源相同则忽略;空表服务由保存前的清洗兜底。
function moveTable(table: string, fromIdx: number, toIdx: number) {
  if (fromIdx === toIdx) return
  removeTable(fromIdx, table)
  const target = partitions.value[toIdx]
  if (target && !target.tables.includes(table)) target.tables.push(table)
}

// 从未分配池把表分给指定服务
function assignTable(table: string, idx: number) {
  const target = partitions.value[idx]
  if (target && !target.tables.includes(table)) target.tables.push(table)
}

function addService() {
  partitions.value.push({serviceName: '', tables: [], description: ''} as ai.MicroservicePartition)
}

// 删除服务:其表退回未分配池(直接从 partitions 移除即可,unassignedTables 会算出来)
function removeService(idx: number) {
  partitions.value.splice(idx, 1)
}

// ==================== 后端生成 Plan/确认 ====================

// 预演生成:文件树落在服务端临时目录,不碰真实项目
async function planBackend() {
  if (planningLoading.value) return null
  planningLoading.value = true
  try {
    await SetGenerateTests(backendWithTests.value)
    const result = await AIPlanBackendCode(ddlContent.value, ormType.value, partitions.value)
    if (result.success) {
      backendPlan.value = result
    }
    return result
  } finally {
    planningLoading.value = false
  }
}

// 确认写入:拷贝预览树进项目并执行后处理链。返回空串表示成功。
async function applyBackend(): Promise<string> {
  const plan = backendPlan.value
  if (!plan || applyingLoading.value) return '生成计划不存在，请重新预览'
  applyingLoading.value = true
  try {
    const err = await AIApplyBackendCode(plan.planId || '')
    if (err === '') {
      backendPlan.value = null
    }
    return err
  } finally {
    applyingLoading.value = false
  }
}

// 放弃预览:清理服务端临时目录
async function discardBackendPlan() {
  const plan = backendPlan.value
  if (!plan) return
  backendPlan.value = null
  try {
    await AIDiscardBackendPlan(plan.planId || '')
  } catch (e) {
    console.error('丢弃生成计划失败:', e)
  }
}

// drawer 预览计划树内的单个文件
async function openPlanFile(path: string) {
  const plan = backendPlan.value
  if (!plan) return null
  const result = await AIReadBackendPlanFile(plan.planId || '', path)
  if (result.success) {
    previewFile.value = path
    previewContent.value = result.content
    previewDrawerOpen.value = true
  }
  return result
}

async function findOpenAPI() {
  if (openapiLoading.value) return null
  openapiLoading.value = true
  try {
    const result = await AIFindOpenAPIFiles()
    if (result.success) {
      openapiFiles.value = result.files || []
      openapiMessage.value = result.message || ''
    }
    return result
  } finally {
    openapiLoading.value = false
  }
}

async function scanForReview() {
  if (reviewScanLoading.value) return null
  reviewScanLoading.value = true
  try {
    const result = await AIReviewScan()
    if (result.success) {
      reviewScanFiles.value = result.files || []
      reviewScanWarnings.value = result.warnings || []
      // 默认全选,用户可勾掉不需要审查的文件
      reviewSelected.value = (result.files || []).map(f => f.path)
    }
    return result
  } finally {
    reviewScanLoading.value = false
  }
}

async function runReview() {
  if (reviewSelected.value.length === 0 || reviewLoading.value) return null
  reviewLoading.value = true
  reviewResult.value = ''
  reviewViewName.value = ''
  reviewBatches.value = []
  reviewBatchTotal.value = 0
  reviewIssues.value = []
  try {
    const result = await AIReviewProject(reviewSelected.value)
    if (result.success) {
      reviewResult.value = result.synthesis || ''
      reviewIssues.value = result.issues || []
    }
    return result
  } finally {
    reviewLoading.value = false
  }
}

// 回看一份历史审查报告(内容载入同一个展示缓冲)
async function viewReview(file: ai.ReviewFile) {
  const result = await LoadAIReviewReport(file.name)
  if (result.success) {
    reviewResult.value = result.content
    reviewViewName.value = file.name
  }
  return result
}

// ==================== 初始化 ====================
let initialized = false

function initAI() {
  bindEvents()
  if (initialized) return
  initialized = true
  checkConfigured()
  const {projectInfo} = useProject()
  // 换项目必须丢掉上一个项目的产物(生成直接写当前项目目录),
  // 并加载新项目里已保存的 .gowind/ai/ 产物恢复现场。
  watch(projectInfo, (pi, prev) => {
    if (pi?.ModPath === prev?.ModPath) return
    resetForProject()
    loadArtifacts()
  }, {immediate: true})
}

// ==================== 对外组合函数 ====================
export function useAI() {
  return {
    // 状态
    mode,
    aiConfigured, checkConfigured,
    requirements, ddlContent, ddlGenerating, ddlAttempt,
    partitions, partitionWarnings, partitionLoading, hasPartitions,
    editingPartitions, ddlTables, unassignedTables, partitionStreamContent,
    ormType, backendPlan, backendWithTests, setBackendWithTests, planningLoading, applyingLoading,
    previewFile, previewContent, previewDrawerOpen,
    openapiFiles, openapiMessage, openapiLoading,
    reviewScanFiles, reviewScanWarnings, reviewScanLoading, reviewSelected,
    reviewLoading, reviewResult, reviewReports, reviewViewName,
    reviewBatches, reviewBatchTotal, reviewIssues,
    // 动作
    initAI, loadArtifacts, resetForProject,
    generateDDL, saveArtifacts, partition, findOpenAPI,
    planBackend, applyBackend, discardBackendPlan, openPlanFile,
    cancelTask,
    beginEditPartitions, cancelEditPartitions, savePartitions,
    removeTable, moveTable, assignTable, addService, removeService,
    scanForReview, runReview, viewReview,
  }
}

import {ref, computed, watch} from 'vue'
import {
  AIGenerateDDLStream,
  AIPartitionMicroservices,
  AIGenerateBackendCode,
  AIValidateDDL,
  AIFindOpenAPIFiles,
  AIReviewScan,
  AIReviewProject,
  LoadAIArtifacts,
  SaveAIArtifact,
  LoadAIReviewReport,
} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime'
import type {ai} from '../../wailsjs/go/models'
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
// 划分编辑态:ddlTables 是 DDL 全表清单(未分配池基准),进入编辑时快照
// partitions,保存或放弃时二选一,避免半途编辑污染生成链的真值。
const editingPartitions = ref(false)
const ddlTables = ref<string[]>([])
let partitionsSnapshot: ai.MicroservicePartition[] | null = null

const ormType = ref<'ent' | 'gorm'>('ent')
const backendGenerating = ref(false)

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

const hasPartitions = computed(() => partitions.value.length > 0)

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

function resetForProject() {
  requirements.value = ''
  ddlContent.value = ''
  ddlGenerating.value = false
  ddlStreamAttempt = 0
  ddlAttempt.value = 1
  partitions.value = []
  partitionWarnings.value = []
  partitionLoading.value = false
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

// 返回空串表示成功,非空为错误信息(Go 侧约定)。
async function generateBackend(): Promise<string> {
  if (backendGenerating.value) return '生成进行中'
  backendGenerating.value = true
  try {
    const err = await AIGenerateBackendCode(ddlContent.value, ormType.value, partitions.value)
    return err || ''
  } finally {
    backendGenerating.value = false
  }
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
  try {
    const result = await AIReviewProject(reviewSelected.value)
    if (result.success) {
      reviewResult.value = result.content
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
    requirements, ddlContent, ddlGenerating, ddlAttempt,
    partitions, partitionWarnings, partitionLoading, hasPartitions,
    editingPartitions, ddlTables, unassignedTables,
    ormType, backendGenerating,
    openapiFiles, openapiMessage, openapiLoading,
    reviewScanFiles, reviewScanWarnings, reviewScanLoading, reviewSelected,
    reviewLoading, reviewResult, reviewReports, reviewViewName,
    // 动作
    initAI, loadArtifacts, resetForProject,
    generateDDL, saveArtifacts, partition, generateBackend, findOpenAPI,
    beginEditPartitions, cancelEditPartitions, savePartitions,
    removeTable, moveTable, assignTable, addService, removeService,
    scanForReview, runReview, viewReview,
  }
}

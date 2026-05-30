<script setup lang="ts">
import {ref} from 'vue'
import {message} from 'ant-design-vue'

import {GenerateFrontendCode} from "../../../wailsjs/go/main/App";
import {SelectFolder} from "../../../wailsjs/go/main/App";

import {parseOpenApiYaml, extractServices, type ParsedService, type OpenApiSpec} from "../../utils/openapi-parser";
import {
  generateAll,
  type GeneratedFile,
  type GenerateFileType,
  type RouterModuleConfig,
} from "../../generators/vue-element";

const confirmLoading = ref(false)

// ==================== 步骤控制 ====================
const currentStep = ref(0)

// ==================== 目标框架 ====================
type TargetFramework = 'vue-element' | 'vue-vben' | 'react'
const targetFramework = ref<TargetFramework>('vue-element')

const frameworkOptions = [
  {label: 'Vue3 Element Plus', value: 'vue-element'},
  {label: 'Vue3 Vben', value: 'vue-vben'},
  {label: 'React', value: 'react'},
]

// ==================== OpenAPI 导入方式 ====================
type ImportSource = 'local' | 'remote' | 'paste'
const importSource = ref<ImportSource>('local')

// 本地文件
const selectedFileName = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

// 远程 URL
const remoteUrl = ref('')
const remoteLoading = ref(false)

// 粘贴内容
const yamlContent = ref('')

// ==================== OpenAPI 数据 ====================
const parsedSpec = ref<OpenApiSpec | null>(null)
const parsedServices = ref<ParsedService[]>([])
const selectedServiceKeys = ref<string[]>([])

// ==================== 生成选项 ====================
const generateOptions = ref({
  outputDir: '',
  generateTypes: ['service', 'composable', 'page', 'drawer', 'router', 'locale'] as GenerateFileType[],
})
const routerModules = ref<RouterModuleConfig[]>([])

// ==================== 生成结果 ====================
const generatedFiles = ref<GeneratedFile[]>([])
const selectedFileIndex = ref(0)

const currentFileContent = ref('')

// ==================== 文件列表过滤 ====================
const activeFileType = ref<string>('all')
const fileTypeOptions = [
  {label: '全部', value: 'all'},
  {label: 'Service', value: 'service'},
  {label: 'Composable', value: 'composable'},
  {label: '页面', value: 'page'},
  {label: '抽屉', value: 'drawer'},
  {label: '路由', value: 'router'},
  {label: '国际化', value: 'locale'},
]

const filteredFiles = ref<GeneratedFile[]>([])

function filterFiles() {
  if (activeFileType.value === 'all') {
    filteredFiles.value = generatedFiles.value
  } else {
    filteredFiles.value = generatedFiles.value.filter(f => f.type === activeFileType.value)
  }
  if (selectedFileIndex.value >= filteredFiles.value.length) {
    selectedFileIndex.value = 0
  }
  updatePreview()
}

function updatePreview() {
  if (filteredFiles.value.length === 0) {
    currentFileContent.value = ''
    return
  }
  const idx = Math.min(selectedFileIndex.value, filteredFiles.value.length - 1)
  currentFileContent.value = filteredFiles.value[idx]?.content || ''
}

function selectFile(index: number) {
  selectedFileIndex.value = index
  updatePreview()
}

// ==================== 本地文件选择 ====================
function handleChooseFile() {
  fileInputRef.value?.click()
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  selectedFileName.value = file.name
  const reader = new FileReader()
  reader.onload = (e) => {
    yamlContent.value = e.target?.result as string || ''
    handleParse()
  }
  reader.readAsText(file)
  // 重置 input 以支持重复选择同一文件
  input.value = ''
}

// ==================== 远程 URL 拉取 ====================
async function handleFetchRemote() {
  if (!remoteUrl.value.trim()) {
    message.warning('请输入 OpenAPI 文档地址')
    return
  }

  remoteLoading.value = true
  try {
    // 尝试直接 fetch
    let response: Response
    try {
      response = await fetch(remoteUrl.value.trim())
    } catch {
      // Wails 环境可能存在 CORS 限制，使用 XMLHttpRequest 作为 fallback
      response = await fetchViaXhr(remoteUrl.value.trim())
    }

    if (!response.ok) {
      message.error(`请求失败: ${response.status} ${response.statusText}`)
      return
    }

    const text = await response.text()
    if (!text.trim()) {
      message.error('获取到的内容为空')
      return
    }

    yamlContent.value = text
    message.success('远程文档加载成功')
    handleParse()
  } catch (e: any) {
    message.error(`拉取失败: ${e.message || e}`)
  } finally {
    remoteLoading.value = false
  }
}

function fetchViaXhr(url: string): Promise<Response> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('GET', url)
    xhr.onload = () => {
      resolve(new Response(xhr.responseText, {
        status: xhr.status,
        statusText: xhr.statusText,
      }))
    }
    xhr.onerror = () => reject(new Error('网络请求失败'))
    xhr.send()
  })
}

// ==================== 解析 ====================
function handleParse() {
  try {
    if (!yamlContent.value.trim()) return
    parsedSpec.value = parseOpenApiYaml(yamlContent.value)
    parsedServices.value = extractServices(parsedSpec.value)

    selectedServiceKeys.value = parsedServices.value
      .filter((s: ParsedService) => s.operations.some((op: { type: string }) => op.type === 'list'))
      .map(s => s.tagName)

    autoDetectRouterModules(parsedServices.value)
    currentStep.value = 1
  } catch (e: any) {
    message.error('解析 OpenAPI 失败: ' + (e.message || e))
    console.error('解析 OpenAPI 失败:', e)
  }
}

// ==================== 预览 ====================
function handlePreview() {
  const selectedServices = parsedServices.value.filter(
    s => selectedServiceKeys.value.includes(s.tagName)
  )
  if (selectedServices.length === 0) return

  if (targetFramework.value === 'vue-element') {
    generatedFiles.value = generateAll({
      services: selectedServices,
      serviceName: '',
      generateTypes: generateOptions.value.generateTypes,
      routerModules: generateOptions.value.generateTypes.includes('router') ? routerModules.value : undefined,
    })
  } else {
    // 其他框架暂未实现，生成占位提示
    generatedFiles.value = selectedServices.map(s => ({
      path: `${targetFramework.value}/${s.tagName}/placeholder.txt`,
      content: `[${frameworkOptions.find(f => f.value === targetFramework.value)?.label}] 代码生成器尚未实现\n\n服务: ${s.modelName}\n描述: ${s.description}\n字段数: ${s.fields.length}\n操作: ${s.operations.map(o => o.type).join(', ')}\n\n敬请期待...`,
      type: 'service' as GenerateFileType,
      description: `${s.modelName} - 待实现`,
      serviceName: s.tagName,
    }))
  }

  selectedFileIndex.value = 0
  activeFileType.value = 'all'
  filterFiles()
  currentStep.value = 2
}

// ==================== 输出目录选择 ====================
async function handleSelectOutputDir() {
  try {
    const folder = await SelectFolder()
    if (folder) {
      generateOptions.value.outputDir = folder
    }
  } catch (e) {
    console.error('选择目录失败:', e)
  }
}

// ==================== 选择 ====================
function handleSelectAll() {
  selectedServiceKeys.value = selectedServiceKeys.value.length === parsedServices.value.length
    ? [] : parsedServices.value.map(s => s.tagName)
}

function handleSelectCrud() {
  selectedServiceKeys.value = parsedServices.value
    .filter((s: ParsedService) => s.operations.some((op: { type: string }) => op.type === 'list'))
    .map(s => s.tagName)
}

// ==================== 确认生成 ====================
async function handleCommit() {
  try {
    confirmLoading.value = true
    const frameworkMap: Record<TargetFramework, string> = {
      'vue-element': 'vue-element',
      'vue-vben': 'vue-vben',
      'react': 'react',
    }
    const res = await GenerateFrontendCode(generateOptions.value.outputDir, frameworkMap[targetFramework.value])
    if (res === '') {
      currentStep.value = 0
      resetState()
    }
  } catch (error) {
    console.error('代码生成失败:', error)
  } finally {
    confirmLoading.value = false
  }
}

function resetState() {
  yamlContent.value = ''
  parsedSpec.value = null
  parsedServices.value = []
  selectedServiceKeys.value = []
  generatedFiles.value = []
  filteredFiles.value = []
  selectedFileIndex.value = 0
  routerModules.value = []
  currentFileContent.value = ''
  selectedFileName.value = ''
  remoteUrl.value = ''
  generateOptions.value.outputDir = ''
}

// ==================== 路由模块自动检测 ====================
function autoDetectRouterModules(services: ParsedService[]) {
  const groupMap = new Map<string, ParsedService[]>()
  for (const service of services) {
    const parts = service.basePath.split('/').filter(Boolean)
    const groupKey = parts.length >= 3 ? parts[2].split('-')[0] : 'other'
    if (!groupMap.has(groupKey)) groupMap.set(groupKey, [])
    groupMap.get(groupKey)!.push(service)
  }

  const moduleIconMap: Record<string, string> = {
    'api': 'lucide:route', 'dict': 'lucide:library-big', 'file': 'lucide:file-search',
    'login': 'lucide:shield-x', 'permission': 'lucide:shield-check', 'opm': 'lucide:users',
    'user': 'lucide:user', 'role': 'lucide:shield-user', 'menu': 'lucide:square-menu',
    'tenant': 'lucide:building-2', 'internal': 'lucide:message-square',
    'audit': 'lucide:scroll-text', 'language': 'lucide:globe', 'task': 'lucide:list-todo',
  }

  routerModules.value = []
  let order = 2001
  for (const [groupKey, groupServices] of groupMap) {
    const moduleKey = ['audit', 'login', 'operation', 'data'].includes(groupKey) ? 'log'
      : ['dict', 'file', 'language', 'task', 'loginP'].includes(groupKey) ? 'system'
      : ['permission', 'role', 'menu'].includes(groupKey) ? 'permission'
      : ['user', 'org', 'position'].includes(groupKey) ? 'opm'
      : groupKey === 'tenant' ? 'tenant'
      : groupKey === 'internal' ? 'internalMessage'
      : groupKey

    const existing = routerModules.value.find(m => m.moduleKey === moduleKey)
    if (existing) {
      existing.serviceTags.push(...groupServices.map(s => s.tagName))
    } else {
      const desc = groupServices[0].description
      const displayName = desc.replace(/管理.*/, '').replace(/服务.*/, '').replace(/查询.*/, '').replace(/日志.*/, '日志审计').trim()
      routerModules.value.push({
        moduleKey,
        moduleDisplayName: displayName || moduleKey,
        moduleIcon: moduleIconMap[groupKey] || 'lucide:folder',
        moduleOrder: order++,
        authority: [],
        serviceTags: groupServices.map(s => s.tagName),
      })
    }
  }
}

function getOperationTag(type: string) {
  const map: Record<string, { color: string; text: string }> = {
    list: {color: 'blue', text: '列表'}, get: {color: 'cyan', text: '详情'},
    create: {color: 'green', text: '创建'}, update: {color: 'orange', text: '更新'},
    delete: {color: 'red', text: '删除'}, other: {color: 'default', text: '其他'},
  }
  return map[type] || map.other
}

function getFileTypeColor(type: string) {
  const map: Record<string, string> = {
    service: 'green', composable: 'cyan', page: 'blue',
    drawer: 'purple', router: 'geekblue', locale: 'gold',
  }
  return map[type] || 'default'
}

function getFrameworkLabel(value: string) {
  return frameworkOptions.find(f => f.value === value)?.label || value
}
</script>

<template>
  <div class="frontend-gen-container">
    <!-- 步骤条 -->
    <a-steps :current="currentStep" size="small" style="margin-bottom: 20px">
      <a-step title="导入 OpenAPI"/>
      <a-step title="生成配置"/>
      <a-step title="预览 &amp; 生成"/>
    </a-steps>

    <!-- ====== 步骤 0: 导入 OpenAPI ====== -->
    <div v-if="currentStep === 0" class="step-content">
      <!-- 目标框架选择 -->
      <a-card title="目标框架" size="small" style="margin-bottom: 16px">
        <a-radio-group v-model:value="targetFramework" button-style="solid">
          <a-radio-button v-for="opt in frameworkOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </a-radio-button>
        </a-radio-group>
      </a-card>

      <!-- 导入方式切换 -->
      <a-card title="导入 OpenAPI 文档" size="small">
        <a-radio-group v-model:value="importSource" style="margin-bottom: 16px">
          <a-radio-button value="local">本地文件</a-radio-button>
          <a-radio-button value="remote">远程地址</a-radio-button>
          <a-radio-button value="paste">粘贴内容</a-radio-button>
        </a-radio-group>

        <!-- 本地文件选择 -->
        <div v-if="importSource === 'local'">
          <input
            ref="fileInputRef"
            type="file"
            accept=".yaml,.yml,.json"
            style="display: none"
            @change="handleFileChange"
          />
          <div class="file-drop-zone" @click="handleChooseFile">
            <div class="drop-zone-content">
              <div style="font-size: 32px; color: #1890ff; margin-bottom: 8px">&#128196;</div>
              <div style="font-weight: 500; margin-bottom: 4px">
                {{ selectedFileName || '点击选择 OpenAPI 文件' }}
              </div>
              <div style="color: #999; font-size: 12px">支持 .yaml / .yml / .json 格式</div>
            </div>
          </div>
        </div>

        <!-- 远程 URL -->
        <div v-if="importSource === 'remote'">
          <a-input-search
            v-model:value="remoteUrl"
            placeholder="输入 OpenAPI 文档 URL，如 https://petstore.swagger.io/v2/swagger.yaml"
            enter-button="拉取"
            :loading="remoteLoading"
            @search="handleFetchRemote"
            style="margin-bottom: 12px"
          />
          <a-alert v-if="!remoteUrl" message="请输入可公开访问的 OpenAPI 文档地址" type="info" show-icon/>
        </div>

        <!-- 粘贴 YAML -->
        <div v-if="importSource === 'paste'">
          <a-textarea
            v-model:value="yamlContent"
            placeholder="请粘贴 OpenAPI 3.0 YAML / JSON 内容..."
            :auto-size="{ minRows: 12, maxRows: 22 }"
            style="font-family: 'Courier New', monospace; font-size: 12px;"
          />
        </div>
      </a-card>

      <div style="text-align: right; margin-top: 16px">
        <a-button type="primary" @click="handleParse" :disabled="!yamlContent.trim()">
          解析 OpenAPI
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 1: 配置生成 ====== -->
    <div v-if="currentStep === 1" class="step-content">
      <!-- 生成配置 -->
      <a-card title="生成配置" size="small" style="margin-bottom: 16px">
        <a-form layout="inline">
          <a-form-item label="目标框架">
            <a-select v-model:value="targetFramework" style="width: 180px">
              <a-select-option v-for="opt in frameworkOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="输出目录">
            <a-input-group compact>
              <a-input
                v-model:value="generateOptions.outputDir"
                placeholder="选择前端项目根目录"
                style="width: calc(100% - 90px)"
                read-only
              />
              <a-button type="primary" @click="handleSelectOutputDir">选择目录</a-button>
            </a-input-group>
          </a-form-item>
          <a-form-item v-if="targetFramework === 'vue-element'" label="生成类型">
            <a-checkbox-group v-model:value="generateOptions.generateTypes">
              <a-checkbox value="service">Service层</a-checkbox>
              <a-checkbox value="composable">Composable层</a-checkbox>
              <a-checkbox value="page">列表页面</a-checkbox>
              <a-checkbox value="drawer">编辑抽屉</a-checkbox>
              <a-checkbox value="router">路由配置</a-checkbox>
              <a-checkbox value="locale">国际化</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
        </a-form>
        <a-alert
          v-if="targetFramework !== 'vue-element'"
          :message="`${getFrameworkLabel(targetFramework)} 代码生成器尚未实现，预览将显示占位内容`"
          type="warning"
          show-icon
          style="margin-top: 12px"
        />
      </a-card>

      <!-- 服务列表 -->
      <a-card size="small">
        <template #title>
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <span>选择要生成的服务 ({{ selectedServiceKeys.length }}/{{ parsedServices.length }})</span>
            <a-space>
              <a-button size="small" @click="handleSelectAll">
                {{ selectedServiceKeys.length === parsedServices.length ? '取消全选' : '全选' }}
              </a-button>
              <a-button size="small" type="dashed" @click="handleSelectCrud">仅选有列表的</a-button>
            </a-space>
          </div>
        </template>

        <a-checkbox-group v-model:value="selectedServiceKeys" style="width: 100%">
          <div v-for="service in parsedServices" :key="service.tagName"
               style="padding: 8px 12px; border-bottom: 1px solid #f0f0f0; display: flex; align-items: center;">
            <a-checkbox :value="service.tagName" style="margin-right: 12px"/>
            <div style="flex: 1">
              <div style="font-weight: 500; margin-bottom: 4px;">
                {{ service.modelName }}
                <span style="color: #999; font-weight: normal; margin-left: 8px; font-size: 12px;">{{ service.description }}</span>
              </div>
              <div style="display: flex; gap: 4px; flex-wrap: wrap;">
                <a-tag v-for="op in service.operations" :key="op.operationId"
                       :color="getOperationTag(op.type).color" size="small">
                  {{ getOperationTag(op.type).text }}
                </a-tag>
                <span style="color: #999; font-size: 12px; margin-left: 8px;">{{ service.fields.length }} 个字段</span>
              </div>
            </div>
          </div>
        </a-checkbox-group>
      </a-card>

      <div style="display: flex; justify-content: space-between; margin-top: 16px">
        <a-button @click="currentStep = 0">上一步</a-button>
        <a-button type="primary" @click="handlePreview" :disabled="selectedServiceKeys.length === 0">
          预览生成代码
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 2: 预览 & 生成 ====== -->
    <div v-if="currentStep === 2" class="step-content">
      <div style="display: flex; gap: 16px; height: calc(100vh - 240px); min-height: 400px;">
        <!-- 左侧文件列表 -->
        <div class="file-list-panel">
          <div class="file-list-header">
            <span>文件 ({{ filteredFiles.length }})</span>
            <a-select v-model:value="activeFileType" :options="fileTypeOptions" size="small"
                      style="width: 110px" @change="filterFiles"/>
          </div>
          <div class="file-list-body">
            <div v-for="(file, index) in filteredFiles" :key="file.path"
                 :class="['file-item', { active: selectedFileIndex === index }]"
                 @click="selectFile(index)">
              <div class="file-path">{{ file.path }}</div>
              <div class="file-meta">
                <a-tag :color="getFileTypeColor(file.type)" size="small">{{ file.type }}</a-tag>
                <span class="file-desc">{{ file.description }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 右侧代码预览 -->
        <div class="code-preview-panel">
          <pre v-if="currentFileContent" class="code-content">{{ currentFileContent }}</pre>
          <a-empty v-else description="没有生成的文件" style="margin-top: 100px"/>
        </div>
      </div>

      <div style="display: flex; justify-content: space-between; margin-top: 16px">
        <a-button @click="currentStep = 1">上一步</a-button>
        <a-space>
          <a-button @click="currentStep = 0; resetState()">重新开始</a-button>
          <a-button type="primary" :loading="confirmLoading" @click="handleCommit">
            确认生成代码
          </a-button>
        </a-space>
      </div>
    </div>
  </div>
</template>

<style scoped>
.frontend-gen-container {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.step-content {
  flex: 1;
  overflow: auto;
}

/* 本地文件选择区域 */
.file-drop-zone {
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  padding: 32px 16px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  background: #fafafa;
}

.file-drop-zone:hover {
  border-color: #1890ff;
  background: #f0f7ff;
}

.drop-zone-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* 文件列表面板 */
.file-list-panel {
  width: 320px;
  flex-shrink: 0;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.file-list-header {
  padding: 8px 12px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 500;
  font-size: 13px;
}

.file-list-body {
  flex: 1;
  overflow-y: auto;
}

.file-item {
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
  border-left: 3px solid transparent;
  cursor: pointer;
  transition: background 0.15s;
}

.file-item:hover {
  background: #f5f5f5;
}

.file-item.active {
  background: #e6f7ff;
  border-left-color: #1890ff;
}

.file-path {
  font-size: 12px;
  font-family: 'Consolas', 'Courier New', monospace;
  word-break: break-all;
  color: #333;
}

.file-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
}

.file-desc {
  font-size: 11px;
  color: #999;
}

/* 代码预览面板 */
.code-preview-panel {
  flex: 1;
  min-width: 0;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  overflow: auto;
  background: #fafafa;
}

.code-content {
  padding: 16px;
  margin: 0;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: #333;
}
</style>

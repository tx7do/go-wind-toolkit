<script setup lang="ts">
import {ref, computed} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  InboxOutlined,
  CloudDownloadOutlined,
  FileTextOutlined,
  SettingOutlined,
  FolderOpenOutlined,
  EyeOutlined,
  CheckSquareOutlined,
  FilterOutlined,
  SendOutlined,
  UndoOutlined,
  RightOutlined,
} from '@ant-design/icons-vue'

import {GenerateFrontendCode} from "../../../wailsjs/go/main/App";
import {SelectFolder} from "../../../wailsjs/go/main/App";

import {parseOpenApiYaml, extractServices, type ParsedService, type OpenApiSpec} from "../../utils/openapi-parser";
import {
  generateAll as generateVueAll,
  type GeneratedFile as VueGeneratedFile,
  type GenerateFileType as VueGenerateFileType,
  type RouterModuleConfig as VueRouterModuleConfig,
} from "../../generators/vue-element";
import {
  generateAll as generateReactAll,
  type ReactGeneratedFile,
  type ReactGenerateFileType,
  type ReactRouterModuleConfig,
} from "../../generators/react-antd";
import {
  generateAll as generateVbenAll,
  type VbenGeneratedFile,
  type VbenGenerateFileType,
  type VbenRouterModuleConfig,
} from "../../generators/vue-vben";
import MonacoEditor from "../backend/MonacoEditor.vue";

const {t} = useI18n()

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
  generateTypes: ['service', 'composable', 'page', 'drawer', 'router', 'locale'] as string[],
})
const routerModules = ref<(VueRouterModuleConfig | ReactRouterModuleConfig | VbenRouterModuleConfig)[]>([])

// ==================== 生成结果 ====================
const generatedFiles = ref<(VueGeneratedFile | ReactGeneratedFile | VbenGeneratedFile)[]>([])
const selectedFileIndex = ref(0)

const currentFileContent = ref('')

// ==================== 文件列表过滤 ====================
const activeFileType = ref<string>('all')
const fileTypeOptions = computed(() => {
  const isVben = targetFramework.value === 'vue-vben'
  const isReact = targetFramework.value === 'react'
  return [
    {label: t('frontend.fileType.all'), value: 'all'},
    {label: 'Service', value: 'service'},
    {label: isReact ? 'Hooks' : 'Composable', value: isReact ? 'hooks' : 'composable'},
    {label: t('frontend.fileType.page'), value: 'page'},
    {label: t('frontend.fileType.drawer'), value: 'drawer'},
    {label: t('frontend.fileType.router'), value: 'router'},
    {label: t('frontend.fileType.locale'), value: 'locale'},
  ]
})

const filteredFiles = ref<(VueGeneratedFile | ReactGeneratedFile | VbenGeneratedFile)[]>([])

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

// 拖拽状态
const fileDragging = ref(false)

function handleFileDragOver(e: DragEvent) {
  e.preventDefault()
  fileDragging.value = true
}

function handleFileDragLeave() {
  fileDragging.value = false
}

function handleFileDrop(e: DragEvent) {
  e.preventDefault()
  fileDragging.value = false

  const file = e.dataTransfer?.files?.[0]
  if (!file) return

  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!ext || !['yaml', 'yml', 'json'].includes(ext)) {
    message.warning(t('frontend.import.fileDragWarning'))
    return
  }

  processOpenApiFile(file)
}

function processOpenApiFile(file: File) {
  selectedFileName.value = file.name
  const reader = new FileReader()
  reader.onload = (e) => {
    yamlContent.value = e.target?.result as string || ''
    handleParse()
  }
  reader.readAsText(file)
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  processOpenApiFile(file)
  input.value = ''
}

// ==================== 远程 URL 拉取 ====================
async function handleFetchRemote() {
  if (!remoteUrl.value.trim()) {
    message.warning(t('frontend.import.remoteUrlRequired'))
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
      message.error(t('frontend.import.requestFailed', {status: response.status, text: response.statusText}))
      return
    }

    const text = await response.text()
    if (!text.trim()) {
      message.error(t('frontend.import.remoteEmpty'))
      return
    }

    yamlContent.value = text
    message.success(t('frontend.import.remoteSuccess'))
    handleParse()
  } catch (e: any) {
    message.error(t('frontend.import.remoteFetchFailed', {error: e.message || e}))
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
    xhr.onerror = () => reject(new Error(t('frontend.import.networkError')))
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
    message.error(t('frontend.import.parseFailed', {msg: e.message || e}))
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
    generatedFiles.value = generateVueAll({
      services: selectedServices,
      serviceName: '',
      generateTypes: generateOptions.value.generateTypes as VueGenerateFileType[],
      routerModules: generateOptions.value.generateTypes.includes('router') ? routerModules.value as VueRouterModuleConfig[] : undefined,
    })
  } else if (targetFramework.value === 'react') {
    // React 使用 hooks 而不是 composable
    const reactTypes = generateOptions.value.generateTypes.map(t =>
      t === 'composable' ? 'hooks' : t
    ) as ReactGenerateFileType[]

    generatedFiles.value = generateReactAll({
      services: selectedServices,
      serviceName: '',
      generateTypes: reactTypes,
      routerModules: reactTypes.includes('router') ? routerModules.value as ReactRouterModuleConfig[] : undefined,
    })
  } else if (targetFramework.value === 'vue-vben') {
    generatedFiles.value = generateVbenAll({
      services: selectedServices,
      serviceName: '',
      generateTypes: generateOptions.value.generateTypes as VbenGenerateFileType[],
      routerModules: generateOptions.value.generateTypes.includes('router') ? routerModules.value as VbenRouterModuleConfig[] : undefined,
    })
  } else {
    // 其他框架暂未实现，生成占位提示
    generatedFiles.value = selectedServices.map(s => ({
      path: `${targetFramework.value}/${s.tagName}/placeholder.txt`,
      content: `[${frameworkOptions.find(f => f.value === targetFramework.value)?.label}] ${t('frontend.placeholder.notImplemented')}\n\n${t('frontend.placeholder.service')}: ${s.modelName}\n${t('frontend.placeholder.description')}: ${s.description}\n${t('frontend.placeholder.fields')}: ${s.fields.length}\n${t('frontend.placeholder.operations')}: ${s.operations.map(o => o.type).join(', ')}\n\n${t('frontend.placeholder.comingSoon')}`,
      type: 'service' as const,
      description: `${s.modelName} - ${t('frontend.placeholder.notImplemented')}`,
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

function toggleServiceSelection(tagName: string) {
  const idx = selectedServiceKeys.value.indexOf(tagName)
  if (idx >= 0) {
    selectedServiceKeys.value.splice(idx, 1)
  } else {
    selectedServiceKeys.value.push(tagName)
  }
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
    list: {color: 'blue', text: t('frontend.operation.list')}, get: {color: 'cyan', text: t('frontend.operation.get')},
    create: {color: 'green', text: t('frontend.operation.create')}, update: {color: 'orange', text: t('frontend.operation.update')},
    delete: {color: 'red', text: t('frontend.operation.delete')}, other: {color: 'default', text: t('frontend.operation.other')},
  }
  return map[type] || map.other
}

function getFileTypeColor(type: string) {
  const map: Record<string, string> = {
    service: 'green', composable: 'cyan', hooks: 'cyan', page: 'blue',
    drawer: 'purple', router: 'geekblue', locale: 'gold',
  }
  return map[type] || 'default'
}

function getFrameworkLabel(value: string) {
  return frameworkOptions.find(f => f.value === value)?.label || value
}

// 根据文件路径推断 Monaco 语言
function detectLanguage(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase() || ''
  const langMap: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    vue: 'html',
    html: 'html',
    css: 'css',
    scss: 'css',
    less: 'css',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    sql: 'sql',
    md: 'markdown',
    xml: 'xml',
  }
  return langMap[ext] || 'plaintext'
}

const previewLanguage = computed(() => {
  if (filteredFiles.value.length === 0) return 'plaintext'
  const file = filteredFiles.value[selectedFileIndex.value]
  return file ? detectLanguage(file.path) : 'plaintext'
})
</script>

<template>
  <div class="frontend-gen-container">
    <!-- 步骤条 -->
    <a-steps :current="currentStep" size="small" style="margin-bottom: 20px">
      <a-step :title="t('frontend.steps.importOpenApi')"/>
      <a-step :title="t('frontend.steps.genConfig')"/>
      <a-step :title="t('frontend.steps.previewGenerate')"/>
    </a-steps>

    <!-- ====== 步骤 0: 导入 OpenAPI ====== -->
    <div v-if="currentStep === 0" class="step-content">
      <!-- 目标框架选择 -->
      <a-card :title="t('frontend.framework.title')" size="small" style="margin-bottom: 16px">
        <a-radio-group v-model:value="targetFramework" button-style="solid">
          <a-radio-button v-for="opt in frameworkOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </a-radio-button>
        </a-radio-group>
      </a-card>

      <!-- 导入方式切换 -->
      <a-card :title="t('frontend.import.title')" size="small">
        <a-radio-group v-model:value="importSource" style="margin-bottom: 16px">
          <a-radio-button value="local">{{ t('frontend.import.local') }}</a-radio-button>
          <a-radio-button value="remote">{{ t('frontend.import.remote') }}</a-radio-button>
          <a-radio-button value="paste">{{ t('frontend.import.paste') }}</a-radio-button>
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
          <div
            class="file-drop-zone"
            :class="{ dragging: fileDragging }"
            @click="handleChooseFile"
            @dragover="handleFileDragOver"
            @dragleave="handleFileDragLeave"
            @drop="handleFileDrop"
          >
            <div class="drop-zone-content">
              <div style="font-size: 32px; color: #1890ff; margin-bottom: 8px"><InboxOutlined/></div>
              <div style="font-weight: 500; margin-bottom: 4px">
                {{ selectedFileName || t('frontend.import.fileDropHint') }}
              </div>
              <div style="color: #999; font-size: 12px">{{ t('frontend.import.fileFormatHint') }}</div>
            </div>
          </div>
        </div>

        <!-- 远程 URL -->
        <div v-if="importSource === 'remote'">
          <a-input-search
            v-model:value="remoteUrl"
            :placeholder="t('frontend.import.remotePlaceholder')"
            :enter-button="t('frontend.import.fetchBtn')"
            :loading="remoteLoading"
            @search="handleFetchRemote"
            style="margin-bottom: 12px"
          />
          <a-alert v-if="!remoteUrl" :message="t('frontend.import.remoteHint')" type="info" show-icon/>
        </div>

        <!-- 粘贴 YAML -->
        <div v-if="importSource === 'paste'">
          <a-textarea
            v-model:value="yamlContent"
            :placeholder="t('frontend.import.pastePlaceholder')"
            :auto-size="{ minRows: 12, maxRows: 22 }"
            style="font-family: 'Courier New', monospace; font-size: 12px;"
          />
        </div>
      </a-card>

      <div class="step-footer" style="justify-content: flex-end">
        <a-button type="primary" @click="handleParse" :disabled="!yamlContent.trim()">
          <RightOutlined style="margin-right: 4px"/> {{ t('frontend.import.parseBtn') }}
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 1: 配置生成 ====== -->
    <div v-if="currentStep === 1" class="step-content">
      <!-- 生成配置 -->
      <a-card :title="t('frontend.config.title')" size="small" style="margin-bottom: 16px">
        <a-form layout="inline">
          <a-form-item :label="t('frontend.framework.selectFramework')">
            <a-select v-model:value="targetFramework" style="width: 180px">
              <a-select-option v-for="opt in frameworkOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="t('frontend.config.outputDir')">
            <a-input-group compact>
              <a-input
                v-model:value="generateOptions.outputDir"
                :placeholder="t('frontend.config.outputDirPlaceholder')"
                style="width: calc(100% - 90px)"
                read-only
              />
              <a-button size="small" type="primary" @click="handleSelectOutputDir"><FolderOpenOutlined style="margin-right: 4px"/> {{ t('frontend.config.selectDir') }}</a-button>
            </a-input-group>
          </a-form-item>
          <a-form-item v-if="targetFramework === 'vue-element' || targetFramework === 'react' || targetFramework === 'vue-vben'" :label="t('frontend.config.generateTypes')">
            <a-checkbox-group v-model:value="generateOptions.generateTypes">
              <a-checkbox value="service">{{ t('frontend.config.serviceLayer') }}</a-checkbox>
              <a-checkbox v-if="targetFramework === 'vue-element' || targetFramework === 'vue-vben'" value="composable">{{ t('frontend.config.composableLayer') }}</a-checkbox>
              <a-checkbox v-if="targetFramework === 'react'" value="composable">React Query Hooks</a-checkbox>
              <a-checkbox value="page">{{ t('frontend.config.listPage') }}</a-checkbox>
              <a-checkbox value="drawer">{{ t('frontend.config.editDrawer') }}</a-checkbox>
              <a-checkbox value="router">{{ t('frontend.config.routerConfig') }}</a-checkbox>
              <a-checkbox value="locale">{{ t('frontend.config.i18n') }}</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
        </a-form>
        <a-alert
          v-if="targetFramework !== 'vue-element' && targetFramework !== 'react' && targetFramework !== 'vue-vben'"
          :message="t('frontend.config.notImplemented', {framework: getFrameworkLabel(targetFramework)})"
          type="warning"
          show-icon
          style="margin-top: 12px"
        />
      </a-card>

      <!-- 服务列表 -->
      <a-card size="small">
        <template #title>
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <span>{{ t('frontend.service.selectTitle', {selected: selectedServiceKeys.length, total: parsedServices.length}) }}</span>
            <a-space>
              <a-button size="small" @click="handleSelectAll">
                {{ selectedServiceKeys.length === parsedServices.length ? t('frontend.service.deselectAll') : t('frontend.service.selectAll') }}
              </a-button>
              <a-button size="small" type="dashed" @click="handleSelectCrud">{{ t('frontend.service.selectListOnly') }}</a-button>
            </a-space>
          </div>
        </template>

        <a-list
          :data-source="parsedServices"
          size="small"
          :split="true"
          class="service-select-list"
        >
          <template #renderItem="{ item: service }">
            <a-list-item
              class="service-list-item"
              :class="{ selected: selectedServiceKeys.includes(service.tagName) }"
              @click="toggleServiceSelection(service.tagName)"
            >
              <a-list-item-meta>
                <template #title>
                  <div class="service-item-title">
                    <span class="service-model-name">{{ service.modelName }}</span>
                    <a-tag v-if="selectedServiceKeys.includes(service.tagName)" color="blue" size="small" style="margin-left: 6px">{{ t('frontend.service.selected') || 'Selected' }}</a-tag>
                  </div>
                </template>
                <template #description>
                  <div class="service-item-desc">
                    <span class="service-desc-text">{{ service.description }}</span>
                    <div class="service-meta-row">
                      <a-tag v-for="op in service.operations" :key="op.operationId"
                             :color="getOperationTag(op.type).color" size="small">
                        {{ getOperationTag(op.type).text }}
                      </a-tag>
                      <span class="service-field-count">{{ t('frontend.service.fields', {count: service.fields.length}) }}</span>
                    </div>
                  </div>
                </template>
                <template #avatar>
                  <a-checkbox
                    :checked="selectedServiceKeys.includes(service.tagName)"
                    @click.stop
                    @change="toggleServiceSelection(service.tagName)"
                  />
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-card>

      <div class="step-footer">
        <a-button @click="currentStep = 0">{{ t('common.prevStep') }}</a-button>
        <a-button type="primary" @click="handlePreview" :disabled="selectedServiceKeys.length === 0">
          <EyeOutlined style="margin-right: 4px"/> {{ t('frontend.preview.previewBtn') }}
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 2: 预览 & 生成 ====== -->
    <div v-if="currentStep === 2" class="step-content">
      <div style="display: flex; gap: 16px; height: calc(100vh - 240px); min-height: 400px;">
        <!-- 左侧文件列表 -->
        <div class="file-list-panel">
          <div class="file-list-header">
            <span>{{ t('frontend.preview.files', {count: filteredFiles.length}) }}</span>
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
          <div v-if="currentFileContent" class="monaco-wrapper">
            <MonacoEditor
              :key="selectedFileIndex + '-' + previewLanguage"
              :model-value="currentFileContent"
              :language="previewLanguage"
              :read-only="true"
              height="100%"
            />
          </div>
          <a-empty v-else :description="t('frontend.preview.noFiles')" style="margin-top: 100px"/>
        </div>
      </div>

      <div class="step-footer">
        <a-button @click="currentStep = 1">{{ t('common.prevStep') }}</a-button>
        <a-space>
          <a-button @click="currentStep = 0; resetState()"><UndoOutlined style="margin-right: 4px"/> {{ t('frontend.preview.resetBtn') }}</a-button>
          <a-button type="primary" :loading="confirmLoading" @click="handleCommit">
            <SendOutlined style="margin-right: 4px"/> {{ t('frontend.preview.confirmBtn') }}
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

.file-drop-zone.dragging {
  border-color: #1890ff;
  background: #e6f7ff;
  box-shadow: 0 0 0 3px rgba(24, 144, 255, 0.1);
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
  overflow: hidden;
  background: #fafafa;
  position: relative;
}

.monaco-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
}

/* 服务选择列表 */
.service-select-list {
  max-height: calc(100vh - 420px);
  overflow-y: auto;
}

.service-list-item {
  cursor: pointer;
  padding: 10px 16px !important;
  transition: background 0.15s, border-color 0.15s;
  border-left: 3px solid transparent;
}

.service-list-item:hover {
  background: #f5f7fa;
}

.service-list-item.selected {
  background: #f0f7ff;
  border-left-color: #1890ff;
}

.service-item-title {
  display: flex;
  align-items: center;
}

.service-model-name {
  font-weight: 600;
  font-size: 14px;
  color: #262626;
}

.service-item-desc {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.service-desc-text {
  font-size: 12px;
  color: #8c8c8c;
}

.service-meta-row {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.service-field-count {
  font-size: 12px;
  color: #8c8c8c;
  margin-left: 4px;
}
</style>

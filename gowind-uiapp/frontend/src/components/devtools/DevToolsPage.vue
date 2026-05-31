<script setup lang="ts">
import {ref, reactive, computed, onMounted, onUnmounted} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  SelectFolder, OpenProject, GetProjectInfo, GetDevServices,
  AddService,
  DevRunService,
  DevBufGenerate, DevEntGenerate,
  DevWireGenerate, DevGoModTidy,
} from '../../../wailsjs/go/main/App'
import {EventsOn} from '../../../wailsjs/runtime'

const {t} = useI18n()

const projectInfo = ref<any>(null)
const services = ref<any[]>([])
const selectedRowKeys = ref<string[]>([])
const loading = ref(false)
const outputText = ref('')

// ==================== 输出管理 ====================
function appendOutput(cmd: string, result: any) {
  const now = new Date().toLocaleTimeString()
  const success = result.success
  let text = `> ${cmd}\n`
  text += `  [${now}] ${success ? 'OK' : 'FAIL'}\n`
  if (result.output) text += result.output + '\n'
  if (result.error) text += 'Error: ' + result.error + '\n'
  text += '\n'
  outputText.value = text + outputText.value
}

// ==================== 项目管理 ====================
async function loadServices() {
  try {
    const list = await GetDevServices()
    services.value = list || []
  } catch (e) {
    services.value = []
  }
}

async function handleOpenProject() {
  try {
    const path = await SelectFolder()
    if (!path) return
    const pi = await OpenProject(path)
    if (!pi || !pi.ModPath) {
      message.error(t('backend.project.noProject'))
      return
    }
    projectInfo.value = pi
    selectedRowKeys.value = []
    await loadServices()
    message.success(t('backend.project.ready'))
  } catch (err) {
    message.error(t('backend.project.openFailed'))
  }
}

// ==================== 添加服务 Modal ====================
const addVisible = ref(false)
const adding = ref(false)
const addForm = reactive({
  serviceName: '',
  servers: ['grpc'] as string[],
  dbClients: ['ent'] as string[],
})

const serverOptions = [
  {label: 'gRPC', value: 'grpc'},
  {label: 'REST/BFF', value: 'rest'},
]
const dbClientOptions = [
  {label: 'Ent', value: 'ent'},
  {label: 'GORM', value: 'gorm'},
  {label: 'Redis', value: 'redis'},
]

function openAddModal() {
  addForm.serviceName = ''
  addForm.servers = ['grpc']
  addForm.dbClients = ['ent']
  addVisible.value = true
}

async function handleAddService() {
  if (!addForm.serviceName.trim()) {
    message.warning(t('devTools.addService.nameRequired'))
    return
  }
  adding.value = true
  try {
    const result = await AddService({
      serviceName: addForm.serviceName,
      servers: addForm.servers,
      dbClients: addForm.dbClients,
    })
    if (result.success) {
      message.success(t('devTools.addService.success'))
      addVisible.value = false
      await loadServices()
    } else {
      message.error(result.error || t('devTools.addService.failed'))
    }
  } catch (e) {
    message.error(t('devTools.addService.failed'))
  } finally {
    adding.value = false
  }
}

// ==================== 命令执行 ====================
async function execCommand(cmdLabel: string, fn: () => Promise<any>) {
  loading.value = true
  try {
    const result = await fn()
    appendOutput(cmdLabel, result)
    if (result.success) {
      message.success(t('devTools.commands.success'))
    } else {
      message.error(result.error || t('devTools.commands.failed'))
    }
  } catch (e: any) {
    appendOutput(cmdLabel, {success: false, error: e.toString()})
    message.error(t('devTools.commands.failed'))
  } finally {
    loading.value = false
  }
}

// -- 单点 --
function handleRunService(name: string) {
  execCommand(`run ${name}`, () => DevRunService(name))
}
function handleEntGenerate(name: string) {
  execCommand(`ent generate ${name}`, () => DevEntGenerate(name))
}
function handleWireGenerate(name: string) {
  execCommand(`wire ${name}`, () => DevWireGenerate(name))
}

// -- 群控 --
function handleBufGenerate() {
  execCommand('buf generate', () => DevBufGenerate())
}
function handleEntGenerateAll() {
  execCommand('ent generate (all)', () => DevEntGenerate(''))
}
function handleWireGenerateAll() {
  execCommand('wire (all)', () => DevWireGenerate(''))
}
function handleGoModTidy() {
  execCommand('go mod tidy', () => DevGoModTidy())
}

// -- 批量 --
const hasSelection = computed(() => selectedRowKeys.value.length > 0)

async function batchExec(cmdLabel: string, fn: (name: string) => Promise<any>) {
  if (selectedRowKeys.value.length === 0) {
    message.warning(t('devTools.commands.noSelection'))
    return
  }
  loading.value = true
  for (const name of selectedRowKeys.value) {
    try {
      const result = await fn(name)
      appendOutput(`${cmdLabel} ${name}`, result)
    } catch (e: any) {
      appendOutput(`${cmdLabel} ${name}`, {success: false, error: e.toString()})
    }
  }
  loading.value = false
  message.success(t('devTools.commands.batchDone'))
}

function handleBatchRun() { batchExec('run', (n) => DevRunService(n)) }
function handleBatchEnt() { batchExec('ent generate', (n) => DevEntGenerate(n)) }
function handleBatchWire() { batchExec('wire', (n) => DevWireGenerate(n)) }

// ==================== 表格选择 ====================
const rowSelection = computed(() => ({
  selectedRowKeys: selectedRowKeys.value,
  onChange: (keys: string[]) => { selectedRowKeys.value = keys },
}))

// ==================== 工具 ====================
function clearOutput() { outputText.value = '' }

function onProjectOpened() {
  GetProjectInfo().then(pi => {
    if (pi && pi.ModPath) {
      projectInfo.value = pi
      selectedRowKeys.value = []
      loadServices()
    }
  }).catch(() => {})
}

onMounted(async () => {
  EventsOn('project-opened', onProjectOpened)
  try {
    const pi = await GetProjectInfo()
    if (pi && pi.ModPath) {
      projectInfo.value = pi
      await loadServices()
    }
  } catch (e) { /* ignore */ }
})

onUnmounted(() => {})
</script>

<template>
  <div class="devtools-page">
    <!-- 顶部：项目操作 + 管理按钮 -->
    <div class="top-bar">
      <a-button type="primary" @click="handleOpenProject">
        {{ projectInfo ? t('backend.project.switchProject') : t('backend.project.clickToOpen') }}
      </a-button>
      <span v-if="projectInfo" class="project-path">{{ projectInfo.ModPath }}</span>
      <span v-else class="project-path project-path--empty">{{ t('devTools.service.noServices') }}</span>

      <div class="spacer"/>

      <a-button size="small" :disabled="!projectInfo" @click="openAddModal">{{ t('devTools.addService.btn') }}</a-button>
      <a-button size="small" @click="loadServices" :disabled="!projectInfo">{{ t('common.refresh') }}</a-button>
    </div>

    <!-- 群控 + 批量按钮栏 -->
    <div class="global-actions" v-if="projectInfo">
      <span class="action-group-label">{{ t('devTools.commands.globalActions') }}</span>
      <a-button :loading="loading" @click="handleBufGenerate">{{ t('devTools.commands.bufGenerate') }}</a-button>
      <a-button :loading="loading" @click="handleEntGenerateAll">{{ t('devTools.commands.entGenerateAll') }}</a-button>
      <a-button :loading="loading" @click="handleWireGenerateAll">{{ t('devTools.commands.wireGenerateAll') }}</a-button>
      <a-button :loading="loading" @click="handleGoModTidy">{{ t('devTools.commands.goModTidy') }}</a-button>

      <a-divider type="vertical" style="height: 24px; margin: 0 8px"/>

      <span class="action-group-label">
        {{ t('devTools.commands.batchActions') }}
        <a-tag v-if="hasSelection" color="blue" style="margin-left: 4px">{{ selectedRowKeys.length }}</a-tag>
      </span>
      <a-button :loading="loading" :disabled="!hasSelection" @click="handleBatchRun">{{ t('devTools.commands.runService') }}</a-button>
      <a-button :loading="loading" :disabled="!hasSelection" @click="handleBatchEnt">{{ t('devTools.commands.entGenerate') }}</a-button>
      <a-button :loading="loading" :disabled="!hasSelection" @click="handleBatchWire">{{ t('devTools.commands.wireGenerate') }}</a-button>
    </div>

    <!-- 服务表格 -->
    <div class="table-section" v-if="projectInfo">
      <a-table
          :data-source="services"
          :row-selection="rowSelection"
          :row-key="(record: any) => record.name"
          :pagination="false"
          :scroll="{ x: 700 }"
          size="small"
          bordered
      >
        <a-table-column dataIndex="name" :title="t('devTools.addService.serviceName')" :width="140"/>
        <a-table-column :title="t('projectManager.overview.hasServer')" :width="70" align="center">
          <template #default="{record}">
            <a-tag v-if="record.hasServer" color="green">OK</a-tag>
            <span v-else class="muted">-</span>
          </template>
        </a-table-column>
        <a-table-column :title="t('projectManager.overview.hasConfig')" :width="70" align="center">
          <template #default="{record}">
            <a-tag v-if="record.hasConfig" color="blue">OK</a-tag>
            <span v-else class="muted">-</span>
          </template>
        </a-table-column>
        <a-table-column :title="t('projectManager.overview.hasEnt')" :width="90" align="center">
          <template #default="{record}">
            <a-tag v-if="record.hasEnt" color="orange">{{ (record.entSchemas || []).length }} schemas</a-tag>
            <span v-else class="muted">-</span>
          </template>
        </a-table-column>
        <a-table-column :title="t('devTools.commands.title')" :width="260">
          <template #default="{record}">
            <div class="row-actions">
              <a-button size="small" type="primary" :loading="loading" :disabled="!record.hasServer" @click="handleRunService(record.name)">{{ t('devTools.commands.runService') }}</a-button>
              <a-button size="small" :loading="loading" :disabled="!record.hasEnt" @click="handleEntGenerate(record.name)">{{ t('devTools.commands.entGenerate') }}</a-button>
              <a-button size="small" :loading="loading" :disabled="!record.hasServer" @click="handleWireGenerate(record.name)">{{ t('devTools.commands.wireGenerate') }}</a-button>
            </div>
          </template>
        </a-table-column>
      </a-table>
    </div>

    <!-- 无项目提示 -->
    <div v-else class="empty-state">
      <a-empty :description="t('devTools.service.noServices')"/>
    </div>

    <!-- 输出面板 -->
    <div class="output-panel" v-if="projectInfo">
      <div class="output-header">
        <span class="panel-title">{{ t('devTools.output.title') }}</span>
        <a-button size="small" type="link" @click="clearOutput" :disabled="!outputText">{{ t('devTools.output.clear') }}</a-button>
      </div>
      <div class="output-content" v-if="outputText"><pre>{{ outputText }}</pre></div>
      <div class="output-content output-empty" v-else>{{ t('devTools.output.empty') }}</div>
    </div>

    <!-- ==================== 添加服务 Modal ==================== -->
    <a-modal v-model:open="addVisible" :title="t('devTools.addService.title')" :confirm-loading="adding" @ok="handleAddService" :width="480">
      <a-form layout="vertical" style="margin-top: 16px">
        <a-form-item :label="t('devTools.addService.serviceName')" required>
          <a-input v-model:value="addForm.serviceName" :placeholder="t('devTools.addService.namePlaceholder')"/>
        </a-form-item>
        <a-form-item :label="t('devTools.addService.servers')">
          <a-checkbox-group v-model:value="addForm.servers">
            <a-checkbox v-for="opt in serverOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
        <a-form-item :label="t('devTools.addService.dbClients')">
          <a-checkbox-group v-model:value="addForm.dbClients">
            <a-checkbox v-for="opt in dbClientOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.devtools-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.project-path { font-size: 13px; color: #595959; }
.project-path--empty { color: #bfbfbf; }
.spacer { flex: 1; }

.global-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.action-group-label {
  font-size: 12px;
  font-weight: 600;
  color: #8c8c8c;
  margin-right: 2px;
  white-space: nowrap;
}

.table-section { flex-shrink: 0; overflow: hidden; }
.table-section :deep(.ant-table-wrapper) { width: 100%; }
.table-section :deep(.ant-table) { font-size: 13px; }
.table-section :deep(.ant-table-cell) { padding: 6px 8px !important; }

.row-actions { display: flex; gap: 4px; flex-wrap: nowrap; }
.muted { color: #d9d9d9; }

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.output-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 120px;
  border-top: 1px solid #f0f0f0;
  margin-top: 8px;
}

.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
  flex-shrink: 0;
}

.panel-title { font-weight: 600; font-size: 13px; color: #262626; }

.output-content {
  flex: 1;
  overflow: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}
.output-content pre { margin: 0; white-space: pre-wrap; word-break: break-all; color: #262626; }
.output-empty { display: flex; align-items: center; justify-content: center; color: #bfbfbf; font-size: 13px; }
</style>

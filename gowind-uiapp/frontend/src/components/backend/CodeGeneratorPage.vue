<script setup lang="ts">
import {ref, reactive, watch, onUnmounted} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  FolderOpenOutlined,
  CloseCircleOutlined,
  AppstoreOutlined,
  InboxOutlined,
  DatabaseOutlined,
  CloudDownloadOutlined,
  EditOutlined,
  LinkOutlined,
  ImportOutlined,
  PlusOutlined,
  FileTextOutlined,
  ThunderboltOutlined,
  TableOutlined,
  RocketOutlined,
  RightOutlined,
  CodeOutlined,
} from '@ant-design/icons-vue'

import {
  EditGeneratorOption,
  GetGeneratorOptions,
  SetGeneratorOption,
  GenerateGrpcCode,
  GenerateRestCode,
  ImportSqlTables,
  ImportDatabaseTables,
  ImportGoSchemaTables,
  GetDBConfig,
  TestDatabaseConnection,
  SelectFolder,
  SetDBConfig,
} from "../../../wailsjs/go/main/App";
import {generator} from "../../../wailsjs/go/models";
import {EventsOn, EventsOff} from "../../../wailsjs/runtime";
import {useProject} from "../../stores/project";

import DatabaseImporterModal from "./DatabaseImporterModal.vue";
import SqlImporterModal from "./SqlImporterModal.vue";

const {t} = useI18n()

// ==================== 步骤控制 ====================
const currentStep = ref(0)

// ==================== 项目信息（全局唯一真值，见 stores/project.ts） ====================
const {
  projectInfo,
  projectError,
  projectLoading,
  selectAndOpenProject,
} = useProject()

// ==================== Schema 导入方式 ====================
type ImportSource = 'database' | 'file' | 'remote' | 'editor' | 'go-schema'
const importSource = ref<ImportSource>('database')

// ent:// 与 gorm:// 数据源只在 ORM 匹配时才成立，故导入成功后锁定步骤 2 的 ORM。
// 真值在后端 dbConfig.dsn 里（refreshTableData 时反查），避免五个导入入口各自维护副本。
const schemaOrmLock = ref<'' | 'ent' | 'gorm'>('')

const openDatabaseImporter = ref(false)
const openSqlImporter = ref(false)

// 数据库导入表单
const dbFormRef = ref()
const dbLoading = ref(false)
const dbTestLoading = ref(false)
const dbFormData = reactive({
  dbType: 'mysql',
  dsn: '',
})
const dbTypes = [
  {value: 'mysql', label: 'MySQL'},
  {value: 'postgresql', label: 'PostgreSQL'},
  {value: 'sqlite', label: 'SQLite'},
  {value: 'oracle', label: 'Oracle'},
]
const dbFormRules = {
  dsn: [
    {required: true, message: () => t('backend.import.dsnRequired'), trigger: 'blur'},
    {min: 5, message: () => t('backend.import.dsnMinLength'), trigger: 'blur'},
  ],
}

async function handleTestConnection() {
  try {
    await dbFormRef.value?.validateFields(['dsn'])
    dbTestLoading.value = true
    const result = await TestDatabaseConnection({
      useDSN: true,
      dsn: dbFormData.dsn,
      type: dbFormData.dbType,
      host: "", port: 0, database: "", username: "", password: "", ssl: false, dbPath: "",
    })
    if (result?.success) {
      message.success(t('backend.import.dbConnectSuccess'))
    } else {
      message.error(result?.message || t('backend.import.dbConnectFailed'))
    }
  } catch (e) {
    console.error('连接测试失败:', e)
  } finally {
    dbTestLoading.value = false
  }
}

async function handleDatabaseImport() {
  try {
    await dbFormRef.value?.validate()
    dbLoading.value = true
    const res = await ImportDatabaseTables({
      useDSN: true,
      dsn: dbFormData.dsn,
      type: dbFormData.dbType,
      host: "", port: 0, database: "", username: "", password: "", ssl: false, dbPath: "",
    })
    if (res !== '') {
      message.error(t('backend.import.dbImportFailed', {msg: res}))
      return
    }
    await SetDBConfig({
      database: "", dbPath: "", host: "", password: "", port: 0, ssl: false, username: "",
      dsn: dbFormData.dsn,
      type: dbFormData.dbType,
      useDSN: true,
    })
    await refreshTableData()
    message.success(t('backend.import.dbImportSuccess'))
  } catch (e) {
    console.error('数据库导入失败:', e)
    message.error(t('backend.import.dbConfigError'))
  } finally {
    dbLoading.value = false
  }
}

// Go 源码 schema（ent:// / gorm://）：不连库，直接解析源码目录下的模型定义。
const goSchemaForm = reactive({
  scheme: 'ent',
  dir: '',
})
const goSchemaLoading = ref(false)

async function handleSelectSchemaDir() {
  const dir = await SelectFolder()
  if (dir) goSchemaForm.dir = dir
}

async function handleGoSchemaImport() {
  if (!goSchemaForm.dir) {
    message.error(t('backend.import.goSchemaDirRequired'))
    return
  }
  goSchemaLoading.value = true
  try {
    const res = await ImportGoSchemaTables(`${goSchemaForm.scheme}://${goSchemaForm.dir}`, goSchemaForm.scheme)
    if (res !== '') {
      message.error(t('backend.import.goSchemaImportFailed', {msg: res}))
      return
    }
    await refreshTableData()
    message.success(t('backend.import.goSchemaImportSuccess'))
  } catch (e) {
    console.error('Go 源码 schema 导入失败:', e)
    message.error(t('backend.import.goSchemaImportFailed', {msg: String(e)}))
  } finally {
    goSchemaLoading.value = false
  }
}

// 本地文件
const selectedFileName = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)
const fileLoading = ref(false)

// 远程 URL
const remoteUrl = ref('')
const remoteLoading = ref(false)

// SQL 编辑器内容
const sqlContent = ref('')

// ==================== 表格数据 ====================
const tableData = ref<Array<generator.Option>>([])
const serviceOptions = reactive<Array<{ label: string; value: string }>>([])
const quickSelectService = ref<string>('')

async function handleQuickSelectService(service: string) {
  tableData.value.forEach(row => {
    row.service = service;
  });
  const opts = await GetGeneratorOptions();
  for (let i = 0; i < opts.length; i++) {
    opts[i].service = service;
  }
  await SetGeneratorOption(opts);
  quickSelectService.value = '';
}

async function handleServiceChange(row: generator.Option) {
  await EditGeneratorOption(row);
}

function filterServiceOption(input: string, option: { label: string; value: string }) {
  return option.label?.toLowerCase().includes(input.toLowerCase()) ?? false
}

async function handleExcludeChange(row: generator.Option) {
  await EditGeneratorOption(row);
}

async function refreshServiceOptions() {
  serviceOptions.length = 0;
  (projectInfo.value?.Services ?? []).forEach(service => {
    serviceOptions.push({label: service, value: service});
  });
}

async function refreshTableData() {
  const opts = await GetGeneratorOptions();
  tableData.value = opts || [];
  await refreshSchemaOrmLock();
}

// 数据源一旦是 Go 源码 scheme，就把 ORM 锁定为对应值并同步回表单：
// 生成阶段 sqlkratos 会强制校验配对，UI 提前锁住才不会让用户在第三步点生成才失败。
async function refreshSchemaOrmLock() {
  let lock: '' | 'ent' | 'gorm' = ''
  try {
    const cfg = await GetDBConfig()
    const dsn = cfg?.dsn ?? ''
    if (dsn.startsWith('ent://')) lock = 'ent'
    else if (dsn.startsWith('gorm://')) lock = 'gorm'
  } catch (e) {
    console.error('读取数据源配置失败:', e)
  }
  schemaOrmLock.value = lock
}

// ==================== 导入操作 ====================

// 本地文件选择
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

async function handleFileDrop(e: DragEvent) {
  e.preventDefault()
  fileDragging.value = false

  const file = e.dataTransfer?.files?.[0]
  if (!file) return

  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!ext || !['sql', 'ddl', 'txt'].includes(ext)) {
    message.warning(t('backend.import.fileDragWarning'))
    return
  }

  await processSqlFile(file)
}

async function processSqlFile(file: File) {
  selectedFileName.value = file.name
  fileLoading.value = true

  try {
    const content = await file.text()
    if (!content.trim()) {
      message.error(t('backend.import.fileEmpty'))
      return
    }
    const res = await ImportSqlTables(content.trim())
    if (res !== '') {
      message.error(t('backend.import.sqlImportFailed', {msg: res}))
      return
    }
    await refreshTableData()
    message.success(t('backend.import.sqlFileImportSuccess', {name: file.name}))
  } catch (e) {
    message.error(t('backend.import.fileReadFailed'))
    console.error(e)
  } finally {
    fileLoading.value = false
  }
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  await processSqlFile(file)
  input.value = ''
}

// 远程 URL 拉取
async function handleFetchRemote() {
  if (!remoteUrl.value.trim()) {
    message.warning(t('backend.import.remoteUrlRequired'))
    return
  }

  remoteLoading.value = true
  try {
    let response: Response
    try {
      response = await fetch(remoteUrl.value.trim())
    } catch {
      response = await fetchViaXhr(remoteUrl.value.trim())
    }

    if (!response.ok) {
      message.error(t('backend.import.requestFailed', {status: response.status, text: response.statusText}))
      return
    }

    const text = await response.text()
    if (!text.trim()) {
      message.error(t('backend.import.remoteEmpty'))
      return
    }

    sqlContent.value = text
    const res = await ImportSqlTables(text.trim())
    if (res !== '') {
      message.error(t('backend.import.sqlImportFailed', {msg: res}))
      return
    }
    await refreshTableData()
    message.success(t('backend.import.remoteImportSuccess'))
  } catch (e) {
    message.error(t('backend.import.remoteFetchFailed', {error: e}))
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
    xhr.onerror = () => reject(new Error(t('backend.import.networkError')))
    xhr.send()
  })
}

// SQL 编辑器导入
async function handleSqlImport() {
  const trimmed = sqlContent.value.trim()
  if (!trimmed) {
    message.warning(t('backend.import.sqlRequired'))
    return
  }
  try {
    const res = await ImportSqlTables(trimmed)
    if (res !== '') {
      message.error(t('backend.import.sqlImportFailed', {msg: res}))
      return
    }
    await refreshTableData()
    message.success(t('backend.import.sqlImportSuccess'))
  } catch (e) {
    message.error(t('backend.import.importFailed'))
    console.error(e)
  }
}

// 打开完整 SQL 编辑器弹窗
function handleOpenSqlEditor() {
  openSqlImporter.value = true
}

// ==================== 生成配置 ====================
const generateConfig = reactive({
  generateGrpc: true,
  generateBff: true,
  ormType: 'ent',
  bffServiceName: 'admin',
  grpcServers: ['grpc'] as string[],
})

const transportOptions = [
  {value: 'grpc', label: 'gRPC'},
  {value: 'rest', label: 'REST'},
  {value: 'websocket', label: 'WebSocket'},
]

const ormTypes = [
  {value: 'ent', label: 'Ent'},
  {value: 'gorm', label: 'GORM'},
]

// 导入 Go 源码 schema 后 ORM 只能跟随数据源，否则生成阶段必然报错。
watch(schemaOrmLock, (v) => {
  if (v) generateConfig.ormType = v
})

const excludedCount = ref(0)
const excludeAll = ref(false)
const protoPackageAll = ref('')

type ProtoPackageStrategy = 'per-table' | 'by-service' | 'custom'
const protoPackageStrategy = ref<ProtoPackageStrategy>('per-table')

function updateTableStats() {
  excludedCount.value = tableData.value.filter(r => r.exclude).length
}

async function handleExcludeAll(checked: boolean) {
  for (const row of tableData.value) {
    row.exclude = checked
  }
  const opts = await GetGeneratorOptions()
  for (let i = 0; i < opts.length; i++) {
    opts[i].exclude = checked
  }
  await SetGeneratorOption(opts)
  updateTableStats()
}

async function handleProtoPackageAll() {
  const val = protoPackageAll.value.trim()
  if (!val) return
  for (const row of tableData.value) {
    row.protoPackage = val
  }
  const opts = await GetGeneratorOptions()
  for (let i = 0; i < opts.length; i++) {
    opts[i].protoPackage = val
  }
  await SetGeneratorOption(opts)
}

// ==================== 生成代码 ====================
const confirmLoading = ref(false)

async function handleGenerate() {
  if (!generateConfig.generateGrpc && !generateConfig.generateBff) {
    message.warning(t('backend.generate.atLeastOne'))
    return
  }
  if (generateConfig.generateGrpc && generateConfig.grpcServers.length === 0) {
    message.warning(t('backend.generate.atLeastOneTransport'))
    return
  }

  confirmLoading.value = true
  try {
    if (generateConfig.generateGrpc) {
      const res = await GenerateGrpcCode(generateConfig.ormType, protoPackageStrategy.value, generateConfig.grpcServers);
      if (res !== '') {
        message.error(t('backend.generate.grpcFailed', {msg: res}));
        return;
      }
      message.success(t('backend.generate.grpcSuccess'));
    }

    if (generateConfig.generateBff) {
      const res = await GenerateRestCode(generateConfig.bffServiceName, protoPackageStrategy.value);
      if (res !== '') {
        message.error(t('backend.generate.bffFailed', {msg: res}));
        return;
      }
      message.success(t('backend.generate.bffSuccess'));
    }
  } catch (error) {
    message.error(t('backend.generate.codeGenFailed'));
  } finally {
    confirmLoading.value = false;
  }
}

// ==================== 步骤流转 ====================
function handleNextFromImport() {
  if (!projectInfo.value) {
    message.warning(t('backend.project.clickToOpen'))
    return
  }
  if (tableData.value.length === 0) {
    message.warning(t('backend.import.importSchemaFirst'))
    return
  }
  updateTableStats();
  currentStep.value = 1;
}

function handleNextFromTableConfig() {
  // 检查是否有未排除但未分配服务的表
  const unassigned = tableData.value.filter(r => !r.exclude && !r.service)
  if (unassigned.length > 0) {
    message.warning(t('backend.table.assignServiceFirst'))
    return
  }
  currentStep.value = 2;
}

// ==================== 事件监听 ====================
// 项目由全局 store 持有：顶栏、模块选择器、其它页面切换项目都只走这一个 watch。
// immediate 覆盖「项目已在本页挂载前打开」的情况（tab 面板是懒挂载的）。
watch(projectInfo, async (pi, prev) => {
  refreshServiceOptions();
  await refreshTableData();
  updateTableStats();
  // 换项目必须回到第一步并清掉上一个项目的 DSN/SQL：后端此时已 CleanOptions，
  // 留着旧表单只会让用户拿 A 项目的配置去生成 B 项目。
  if (!pi || pi.ModPath !== prev?.ModPath) {
    currentStep.value = 0;
    dbFormData.dsn = '';
    goSchemaForm.dir = '';
    sqlContent.value = '';
    selectedFileName.value = '';
  }
}, {immediate: true});

EventsOn('table-imported', () => {
  refreshTableData().then(() => {
    updateTableStats();
    if (tableData.value.length > 0 && projectInfo.value) {
      currentStep.value = 1;
    }
  });
})

onUnmounted(() => {
  EventsOff('table-imported')
})
</script>

<template>
  <div class="backend-gen-container">
    <!-- 步骤条 -->
    <a-steps :current="currentStep" size="small" style="margin-bottom: 20px">
      <a-step :title="t('backend.steps.importSchema')"/>
      <a-step :title="t('backend.steps.tableConfig')"/>
      <a-step :title="t('backend.steps.generateConfig')"/>
    </a-steps>

    <!-- ====== 步骤 0: 导入 Schema ====== -->
    <div v-if="currentStep === 0" class="step-content">
      <!-- 打开项目 - 空状态 -->
      <div v-if="!projectInfo && !projectError" class="project-empty-card" @click="selectAndOpenProject">
        <div class="project-empty-icon">
          <FolderOpenOutlined style="font-size: 40px; color: #1890ff"/>
        </div>
        <div v-if="projectLoading" style="font-weight: 500; color: #1890ff">
          <a-spin size="small"/> {{ t('backend.project.identifying') }}
        </div>
        <template v-else>
          <div class="project-empty-title">{{ t('backend.project.clickToOpen') }}</div>
          <div class="project-empty-desc">{{ t('backend.project.selectGoProject') }}</div>
        </template>
      </div>

      <!-- 打开项目 - 错误状态 -->
      <div v-else-if="projectError" class="project-error-card">
        <div class="project-error-left">
          <div class="project-error-indicator">
            <span class="project-error-icon">
              <CloseCircleOutlined style="font-size: 16px; color: #ff4d4f"/>
            </span>
            <span class="project-error-label">{{ t('backend.project.failed') }}</span>
          </div>
          <div class="project-error-msg">{{ projectError }}</div>
          <div class="project-error-hint">{{ t('backend.project.hintGoMod') }}</div>
        </div>
        <a-button size="small" type="primary" @click="selectAndOpenProject">{{ t('backend.project.retry') }}</a-button>
      </div>

      <!-- 项目已打开：精简为一条内联提示，详细信息见顶栏 -->
      <div v-if="projectInfo" class="project-inline">
        <span class="project-opened-dot"></span>
        <span class="project-inline-name">{{ projectInfo.ModPath }}</span>
        <span class="project-inline-meta">
          Go {{ projectInfo.GoVersion }} · {{ t('backend.project.services', {count: projectInfo.Services?.length ?? 0}) }}
        </span>
        <span class="switch-project-link" @click="selectAndOpenProject">{{ t('backend.project.switchProject') }}</span>
      </div>

      <!-- 导入方式 -->
      <a-card :title="t('backend.import.title')" size="small">
        <a-radio-group v-model:value="importSource" style="margin-bottom: 16px">
          <a-radio-button value="database"><DatabaseOutlined style="margin-right: 4px"/> {{ t('backend.import.database') }}</a-radio-button>
          <a-radio-button value="file"><FileTextOutlined style="margin-right: 4px"/> {{ t('backend.import.file') }}</a-radio-button>
          <a-radio-button value="remote"><CloudDownloadOutlined style="margin-right: 4px"/> {{ t('backend.import.remote') }}</a-radio-button>
          <a-radio-button value="editor"><EditOutlined style="margin-right: 4px"/> {{ t('backend.import.editor') }}</a-radio-button>
          <a-radio-button value="go-schema"><CodeOutlined style="margin-right: 4px"/> {{ t('backend.import.goSchema') }}</a-radio-button>
        </a-radio-group>

        <!-- 数据库导入 -->
        <div v-if="importSource === 'database'">
          <a-form
            ref="dbFormRef"
            :model="dbFormData"
            :rules="dbFormRules"
            layout="vertical"
          >
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item :label="t('backend.import.dbType')" name="dbType">
                  <a-select v-model:value="dbFormData.dbType">
                    <a-select-option v-for="db in dbTypes" :key="db.value" :value="db.value">
                      {{ db.label }}
                    </a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="16">
                <a-form-item :label="t('backend.import.dsn')" name="dsn">
                  <a-textarea
                    v-model:value="dbFormData.dsn"
                    :placeholder="t('backend.import.dsnPlaceholder')"
                    :rows="2"
                  />
                </a-form-item>
              </a-col>
            </a-row>
            <div style="display: flex; gap: 8px">
              <a-button type="default" @click="handleTestConnection" :loading="dbTestLoading">
                {{ t('backend.import.testConnection') }}
              </a-button>
              <a-button type="primary" @click="handleDatabaseImport" :loading="dbLoading">
                <ImportOutlined style="margin-right: 4px"/> {{ t('backend.import.importTables') }}
              </a-button>
            </div>
          </a-form>
        </div>

        <!-- 本地文件 -->
        <div v-if="importSource === 'file'">
          <input
            ref="fileInputRef"
            type="file"
            accept=".sql,.ddl"
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
              <div v-if="fileLoading" style="font-weight: 500">
                <a-spin size="small"/> {{ t('common.importing') }}
              </div>
              <div v-else style="font-weight: 500; margin-bottom: 4px">
                {{ selectedFileName || t('backend.import.fileDropHint') }}
              </div>
              <div style="color: #999; font-size: 12px">{{ t('backend.import.fileFormatHint') }}</div>
            </div>
          </div>
        </div>

        <!-- 远程 URL -->
        <div v-if="importSource === 'remote'">
          <a-input-search
            v-model:value="remoteUrl"
            :placeholder="t('backend.import.remotePlaceholder')"
            :enter-button="t('backend.import.fetchBtn')"
            :loading="remoteLoading"
            @search="handleFetchRemote"
            style="margin-bottom: 12px"
          />
          <a-alert v-if="!remoteUrl" :message="t('backend.import.remoteHint')" type="info" show-icon/>
        </div>

        <!-- SQL 编辑器 -->
        <div v-if="importSource === 'editor'">
          <a-textarea
            v-model:value="sqlContent"
            :placeholder="t('backend.import.sqlPlaceholder')"
            :auto-size="{ minRows: 10, maxRows: 20 }"
            style="font-family: 'Courier New', monospace; font-size: 12px; margin-bottom: 12px;"
          />
          <div style="display: flex; gap: 8px">
            <a-button type="primary" @click="handleSqlImport" :disabled="!sqlContent.trim()">
              <ImportOutlined style="margin-right: 4px"/> {{ t('backend.import.importSql') }}
            </a-button>
            <a-button type="default" @click="handleOpenSqlEditor">
              <EditOutlined style="margin-right: 4px"/> {{ t('backend.import.openAdvancedEditor') }}
            </a-button>
          </div>
        </div>

        <!-- Go 源码 schema：不连库，直接解析 ent schema / gorm model 目录 -->
        <div v-if="importSource === 'go-schema'">
          <a-alert :message="t('backend.import.goSchemaHint')" type="info" show-icon style="margin-bottom: 12px"/>
          <div style="display: flex; gap: 16px; margin-bottom: 12px">
            <div>
              <div style="color: #666; font-size: 12px; margin-bottom: 4px">{{ t('backend.import.goScheme') }}</div>
              <a-select v-model:value="goSchemaForm.scheme" style="width: 110px">
                <a-select-option value="ent">ent://</a-select-option>
                <a-select-option value="gorm">gorm://</a-select-option>
              </a-select>
            </div>
            <div style="flex: 1">
              <div style="color: #666; font-size: 12px; margin-bottom: 4px">{{ t('backend.import.schemaDir') }}</div>
              <a-input-group compact>
                <a-input
                  v-model:value="goSchemaForm.dir"
                  :placeholder="goSchemaForm.scheme === 'ent' ? t('backend.import.entDirPlaceholder') : t('backend.import.gormDirPlaceholder')"
                  style="width: calc(100% - 96px)"
                />
                <a-button style="width: 96px" @click="handleSelectSchemaDir">
                  <FolderOpenOutlined style="margin-right: 4px"/> {{ t('backend.import.selectDir') }}
                </a-button>
              </a-input-group>
            </div>
          </div>
          <div style="display: flex; gap: 8px; align-items: center">
            <a-button type="primary" :loading="goSchemaLoading" :disabled="!goSchemaForm.dir" @click="handleGoSchemaImport">
              <ImportOutlined style="margin-right: 4px"/> {{ t('backend.import.importTables') }}
            </a-button>
            <span v-if="goSchemaForm.dir" style="color: #999; font-size: 12px; word-break: break-all">
              {{ goSchemaForm.scheme }}://{{ goSchemaForm.dir }}
            </span>
          </div>
        </div>

        <!-- 已导入提示 -->
        <div v-if="tableData.length > 0" style="margin-top: 16px; padding-top: 12px; border-top: 1px solid #f0f0f0;">
          <a-tag color="success">{{ t('backend.import.importedTables', {count: tableData.length}) }}</a-tag>
          <a-button type="link" size="small" @click="refreshTableData" style="margin-left: 8px">{{ t('common.refresh') }}</a-button>
        </div>
      </a-card>

      <div class="step-footer" style="justify-content: flex-end">
        <a-button type="primary" @click="handleNextFromImport" :disabled="!projectInfo || tableData.length === 0">
          <RightOutlined style="margin-right: 4px"/> {{ t('backend.import.nextStepConfig') }}
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 1: 表配置 ====== -->
    <div v-if="currentStep === 1" class="step-content">
      <!-- Proto 包策略 -->
      <div class="proto-strategy-bar">
        <span class="proto-strategy-label">{{ t('backend.table.protoPackageStrategy') }}</span>
        <a-radio-group v-model:value="protoPackageStrategy" size="small" class="proto-strategy-group">
          <a-tooltip :title="t('backend.table.protoStrategyPerTableTip')">
            <a-radio-button value="per-table">
              <TableOutlined style="margin-right: 4px"/>
              {{ t('backend.table.protoStrategyPerTable') }}
            </a-radio-button>
          </a-tooltip>
          <a-tooltip :title="t('backend.table.protoStrategyByServiceTip')">
            <a-radio-button value="by-service">
              <AppstoreOutlined style="margin-right: 4px"/>
              {{ t('backend.table.protoStrategyByService') }}
            </a-radio-button>
          </a-tooltip>
          <a-tooltip :title="t('backend.table.protoStrategyCustomTip')">
            <a-radio-button value="custom">
              <EditOutlined style="margin-right: 4px"/>
              {{ t('backend.table.protoStrategyCustom') }}
            </a-radio-button>
          </a-tooltip>
        </a-radio-group>
      </div>

      <a-card size="small">
        <template #title>
          <span>{{ t('backend.table.tableCount', {total: tableData.length, excluded: excludedCount}) }}</span>
        </template>
        <template #extra>
          <a-space>
            <a-button size="small" type="primary" ghost @click="openDatabaseImporter = true"><PlusOutlined style="margin-right: 4px"/>{{ t('backend.table.appendImport') }}</a-button>
            <a-button size="small" @click="openSqlImporter = true"><EditOutlined style="margin-right: 4px"/>{{ t('backend.table.sqlImport') }}</a-button>
          </a-space>
        </template>

        <vxe-table
          :data="tableData"
          :row-config="{ keyField: 'id' }"
          size="small"
          class="table-content"
        >
          <vxe-column field="tableName" :title="t('backend.table.tableName')" min-width="200"/>
          <vxe-column v-if="protoPackageStrategy === 'custom'" field="protoPackage" :title="t('backend.table.protoPackage')" min-width="200">
            <template #header>
              <div class="service-header">
                <span>{{ t('backend.table.protoPackage') }}</span>
                <a-input
                  v-model:value="protoPackageAll"
                  :placeholder="t('backend.table.protoPackageAllPlaceholder')"
                  style="width: 140px; margin-left: 8px"
                  size="small"
                  allow-clear
                  @pressEnter="handleProtoPackageAll"
                />
              </div>
            </template>
            <template #default="{ row }">
              <a-input
                v-model:value="row.protoPackage"
                :placeholder="t('backend.table.protoPackagePlaceholder')"
                size="small"
                @change="handleServiceChange(row)"
              />
            </template>
          </vxe-column>
          <vxe-column field="service" :title="t('backend.table.service')" min-width="180">
            <template #header>
              <div class="service-header">
                <span>{{ t('backend.table.service') }}</span>
                <a-select
                  v-model:value="quickSelectService"
                  :options="serviceOptions"
                  :placeholder="t('backend.table.quickSelect')"
                  style="width: 150px; margin-left: 8px"
                  @change="handleQuickSelectService"
                  allow-clear
                />
              </div>
            </template>
            <template #default="{ row }">
              <a-auto-complete
                v-model:value="row.service"
                :options="serviceOptions"
                :placeholder="t('backend.table.selectService')"
                style="width: 100%"
                @change="handleServiceChange(row)"
                allow-clear
                :filter-option="filterServiceOption"
              />
            </template>
          </vxe-column>
          <vxe-column field="exclude" :title="t('backend.table.exclude')" width="100" align="center">
            <template #header>
              <a-switch
                v-model:checked="excludeAll"
                size="small"
                :style="{ backgroundColor: excludeAll ? '#ff4d4f' : undefined }"
                @change="handleExcludeAll"
              />
            </template>
            <template #default="{ row }">
              <a-switch
                v-model:checked="row.exclude"
                :style="{ backgroundColor: row.exclude ? '#ff4d4f' : undefined }"
                @change="handleExcludeChange(row); updateTableStats()"
              />
            </template>
          </vxe-column>
        </vxe-table>
      </a-card>

      <div class="step-footer">
        <a-button @click="currentStep = 0">{{ t('common.prevStep') }}</a-button>
        <a-button type="primary" @click="handleNextFromTableConfig">
          {{ t('backend.generate.nextStepGenerate') }}
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 2: 生成配置 ====== -->
    <div v-if="currentStep === 2" class="step-content">
      <!-- 生成目标 -->
      <a-card :title="t('backend.generate.title')" size="small" style="margin-bottom: 16px">
        <div style="display: flex; flex-direction: column; gap: 16px">
          <!-- gRPC -->
          <div class="target-card" :class="{ active: generateConfig.generateGrpc }">
            <div class="target-header">
              <a-checkbox v-model:checked="generateConfig.generateGrpc">
                <span class="target-title">{{ t('backend.generate.grpcService') }}</span>
              </a-checkbox>
              <a-tag color="blue" size="small">gRPC</a-tag>
            </div>
            <div v-if="generateConfig.generateGrpc" class="target-body">
              <a-form layout="inline">
                <a-form-item :label="t('backend.generate.ormType')">
                  <a-tooltip :title="schemaOrmLock ? t('backend.generate.ormLockedTip', {scheme: `${schemaOrmLock}://`}) : ''">
                    <a-select v-model:value="generateConfig.ormType" style="width: 120px" :disabled="!!schemaOrmLock">
                      <a-select-option v-for="item in ormTypes" :key="item.value" :value="item.value">
                        {{ item.label }}
                      </a-select-option>
                    </a-select>
                  </a-tooltip>
                  <a-tag v-if="schemaOrmLock" color="purple" style="margin-left: 8px">{{ schemaOrmLock }}://</a-tag>
                </a-form-item>
                <a-form-item :label="t('backend.generate.servers')">
                  <a-checkbox-group v-model:value="generateConfig.grpcServers">
                    <a-checkbox v-for="opt in transportOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-checkbox>
                  </a-checkbox-group>
                </a-form-item>
              </a-form>
            </div>
          </div>

          <!-- BFF -->
          <div class="target-card" :class="{ active: generateConfig.generateBff }">
            <div class="target-header">
              <a-checkbox v-model:checked="generateConfig.generateBff">
                <span class="target-title">{{ t('backend.generate.bffService') }}</span>
              </a-checkbox>
              <a-tag color="green" size="small">REST</a-tag>
            </div>
            <div v-if="generateConfig.generateBff" class="target-body">
              <a-form layout="inline">
                <a-form-item :label="t('backend.generate.bffServiceName')">
                  <a-input v-model:value="generateConfig.bffServiceName" style="width: 180px" :placeholder="t('backend.generate.bffServiceNamePlaceholder')"/>
                </a-form-item>
              </a-form>
            </div>
          </div>
        </div>
      </a-card>

      <!-- 生成概览 -->
      <a-card :title="t('backend.generate.summary')" size="small">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item :label="t('backend.generate.project')">{{ projectInfo?.ModPath || '-' }}</a-descriptions-item>
          <a-descriptions-item :label="t('backend.generate.validTables')">{{ tableData.length - excludedCount }} / {{ tableData.length }}</a-descriptions-item>
          <a-descriptions-item :label="t('backend.generate.genGrpc')">{{ generateConfig.generateGrpc ? generateConfig.ormType + ' ORM' : t('backend.generate.no') }}</a-descriptions-item>
          <a-descriptions-item :label="t('backend.generate.genBff')">{{ generateConfig.generateBff ? generateConfig.bffServiceName : t('backend.generate.no') }}</a-descriptions-item>
        </a-descriptions>
      </a-card>

      <div class="step-footer">
        <a-button @click="currentStep = 1">{{ t('common.prevStep') }}</a-button>
          <a-button
          type="primary"
          danger
          :loading="confirmLoading"
          :disabled="!generateConfig.generateGrpc && !generateConfig.generateBff"
          @click="handleGenerate"
        >
          <RocketOutlined style="margin-right: 4px"/> {{ t('backend.generate.startGenerate') }}
        </a-button>
      </div>
    </div>
  </div>

  <!-- 弹窗 -->
  <DatabaseImporterModal v-model:open="openDatabaseImporter"/>
  <SqlImporterModal v-model:open="openSqlImporter"/>
</template>

<style scoped>
.backend-gen-container {
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

/* 项目未打开 - 空状态卡片 */
.project-empty-card {
  border: 2px dashed #d9d9d9;
  border-radius: 10px;
  padding: 28px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  background: #fafafa;
  margin-bottom: 16px;
}

.project-empty-card:hover {
  border-color: #1890ff;
  background: #f0f7ff;
}

.project-empty-card:hover .project-empty-title {
  color: #1890ff;
}

.project-empty-icon {
  margin-bottom: 10px;
}

.project-empty-title {
  font-size: 15px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
  transition: color 0.3s;
}

.project-empty-desc {
  font-size: 12px;
  color: #8c8c8c;
}

/* 项目已打开 - 内联提示条 */
.project-inline {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: 8px;
  padding: 8px 14px;
  margin-bottom: 16px;
}

.project-inline-name {
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
  font-family: 'Consolas', 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-inline-meta {
  color: #595959;
  font-size: 12px;
  white-space: nowrap;
  margin-left: auto;
}

.project-opened-dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #52c41a;
  box-shadow: 0 0 0 3px rgba(82, 196, 26, 0.2);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(82, 196, 26, 0.2); }
  50% { box-shadow: 0 0 0 6px rgba(82, 196, 26, 0.1); }
}

.switch-project-link {
  font-size: 12px;
  color: #595959;
  cursor: pointer;
  white-space: nowrap;
  padding: 4px 12px;
  border-radius: 4px;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
  background: rgba(255, 255, 255, 0.5);
}

.switch-project-link:hover {
  color: #389e0d;
  border-color: #b7eb8f;
  background: rgba(255, 255, 255, 0.8);
}

/* 项目错误卡片 */
.project-error-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff2f0;
  border: 1px solid #ffccc7;
  border-radius: 10px;
  padding: 16px 20px;
  margin-bottom: 16px;
}

.project-error-left {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.project-error-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.project-error-label {
  font-size: 13px;
  font-weight: 600;
  color: #cf1322;
}

.project-error-msg {
  font-size: 13px;
  color: #595959;
}

.project-error-hint {
  font-size: 12px;
  color: #8c8c8c;
}

/* 文件选择区域 */
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

/* 表格 */
.service-header {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
}

:deep(.ant-switch-checked) {
  background-color: #ff4d4f !important;
}

/* 生成目标卡片 */
.target-card {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 12px 16px;
  transition: all 0.2s;
}

.target-card.active {
  border-color: #1890ff;
  background: #f0f7ff;
}

.target-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.target-title {
  font-weight: 500;
  font-size: 14px;
}

.target-body {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

/* Proto 包策略栏 */
.proto-strategy-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #f6f8fc 0%, #eef2f9 100%);
  border: 1px solid #d9e3f0;
  border-radius: 8px;
}

.proto-strategy-label {
  font-size: 13px;
  font-weight: 600;
  color: #4a5568;
  white-space: nowrap;
  letter-spacing: 0.3px;
}

.proto-strategy-group :deep(.ant-radio-button-wrapper) {
  border-radius: 6px !important;
  border: 1px solid #d9d9d9 !important;
  margin-right: 6px;
  padding: 0 14px;
  height: 30px;
  line-height: 28px;
  font-size: 12px;
  transition: all 0.25s;
}

.proto-strategy-group :deep(.ant-radio-button-wrapper-checked) {
  border-color: #1890ff !important;
  background: #1890ff !important;
  color: #fff !important;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.35);
}

.proto-strategy-group :deep(.ant-radio-button-wrapper:not(:first-child)::before) {
  display: none;
}
</style>

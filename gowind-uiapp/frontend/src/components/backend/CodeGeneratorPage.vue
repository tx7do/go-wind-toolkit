<script setup lang="ts">
import {ref, reactive} from 'vue'
import {message} from 'ant-design-vue'

import {
  EditGeneratorOption,
  GetGeneratorOptions,
  GetProjectInfo,
  SetGeneratorOption,
  OpenProject,
  SelectFolder,
  GenerateGrpcCode,
  GenerateRestCode,
  ImportSqlTables,
  ImportDatabaseTables,
  TestDatabaseConnection,
  SetDBConfig,
} from "../../../wailsjs/go/main/App";
import {generator, detect} from "../../../wailsjs/go/models";
import {EventsOn} from "../../../wailsjs/runtime";

import DatabaseImporterModal from "./DatabaseImporterModal.vue";
import SqlImporterModal from "./SqlImporterModal.vue";

// ==================== 步骤控制 ====================
const currentStep = ref(0)

// ==================== 项目信息 ====================
const projectInfo = ref<detect.ProjectInfo>()
const projectError = ref('')
const projectLoading = ref(false)

async function handleOpenProject() {
  try {
    const path = await SelectFolder();
    if (!path) return

    projectLoading.value = true
    projectError.value = ''

    try {
      const pi = await OpenProject(path);
      if (!pi || !pi.ModPath) {
        projectError.value = '所选目录不是一个有效的 Go 项目，未找到 go.mod 文件'
        projectInfo.value = undefined
        return
      }
      projectInfo.value = pi;
      await refreshServiceOptions();
      await refreshTableData();
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      projectError.value = msg || '无法打开项目，请确认目录正确'
      projectInfo.value = undefined
    }
  } catch (err) {
    console.error('选择文件夹出错：', err);
  } finally {
    projectLoading.value = false
  }
}

// ==================== Schema 导入方式 ====================
type ImportSource = 'database' | 'file' | 'remote' | 'editor'
const importSource = ref<ImportSource>('database')

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
    {required: true, message: '请输入数据源名称(DSN)', trigger: 'blur'},
    {min: 5, message: 'DSN长度至少5个字符', trigger: 'blur'},
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
      message.success('数据库连接成功！')
    } else {
      message.error(result?.message || '数据库连接失败')
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
      message.error('数据库导入失败: ' + res)
      return
    }
    await SetDBConfig({
      database: "", dbPath: "", host: "", password: "", port: 0, ssl: false, username: "",
      dsn: dbFormData.dsn,
      type: dbFormData.dbType,
      useDSN: true,
    })
    await refreshTableData()
    message.success('数据库导入成功')
  } catch (e) {
    console.error('数据库导入失败:', e)
    message.error('请检查数据库配置')
  } finally {
    dbLoading.value = false
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

async function handleExcludeChange(row: generator.Option) {
  await EditGeneratorOption(row);
}

async function refreshServiceOptions() {
  const pi = await GetProjectInfo();
  if (pi && pi.Services) {
    serviceOptions.length = 0;
    pi.Services.forEach(service => {
      serviceOptions.push({label: service, value: service});
    });
  }
}

async function refreshTableData() {
  const opts = await GetGeneratorOptions();
  tableData.value = opts || [];
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
    message.warning('请拖入 .sql 或 .ddl 格式的文件')
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
      message.error('文件内容为空')
      return
    }
    const res = await ImportSqlTables(content.trim())
    if (res !== '') {
      message.error('SQL 导入失败: ' + res)
      return
    }
    await refreshTableData()
    message.success(`从文件 ${file.name} 成功导入表结构`)
  } catch (e) {
    message.error('文件读取失败')
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
    message.warning('请输入 SQL 文件地址')
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
      message.error(`请求失败: ${response.status} ${response.statusText}`)
      return
    }

    const text = await response.text()
    if (!text.trim()) {
      message.error('获取到的内容为空')
      return
    }

    sqlContent.value = text
    const res = await ImportSqlTables(text.trim())
    if (res !== '') {
      message.error('SQL 导入失败: ' + res)
      return
    }
    await refreshTableData()
    message.success('远程 SQL 导入成功')
  } catch (e) {
    message.error(`拉取失败: ${e}`)
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

// SQL 编辑器导入
async function handleSqlImport() {
  const trimmed = sqlContent.value.trim()
  if (!trimmed) {
    message.warning('请输入 SQL DDL 语句')
    return
  }
  try {
    const res = await ImportSqlTables(trimmed)
    if (res !== '') {
      message.error('SQL 导入失败: ' + res)
      return
    }
    await refreshTableData()
    message.success('SQL 导入成功')
  } catch (e) {
    message.error('导入失败')
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
})

const ormTypes = [
  {value: 'ent', label: 'Ent'},
  {value: 'gorm', label: 'GORM'},
]

const excludedCount = ref(0)

function updateTableStats() {
  excludedCount.value = tableData.value.filter(r => r.exclude).length
}

// ==================== 生成代码 ====================
const confirmLoading = ref(false)

async function handleGenerate() {
  if (!generateConfig.generateGrpc && !generateConfig.generateBff) {
    message.warning('请至少选择一种生成目标')
    return
  }

  confirmLoading.value = true
  try {
    if (generateConfig.generateGrpc) {
      const res = await GenerateGrpcCode(generateConfig.ormType);
      if (res !== '') {
        message.error('gRPC 代码生成失败: ' + res);
        return;
      }
      message.success('gRPC 服务代码生成成功');
    }

    if (generateConfig.generateBff) {
      const res = await GenerateRestCode(generateConfig.bffServiceName);
      if (res !== '') {
        message.error('BFF 代码生成失败: ' + res);
        return;
      }
      message.success('BFF 服务代码生成成功');
    }
  } catch (error) {
    message.error('代码生成失败');
  } finally {
    confirmLoading.value = false;
  }
}

// ==================== 步骤流转 ====================
function handleNextFromImport() {
  if (tableData.value.length === 0) {
    message.warning('请先导入 Schema');
    return;
  }
  updateTableStats();
  currentStep.value = 1;
}

// ==================== 事件监听 ====================
EventsOn('project-opened', () => {
  refreshServiceOptions();
  GetProjectInfo().then(pi => {
    if (pi) projectInfo.value = pi;
  });
})

EventsOn('table-imported', () => {
  refreshTableData().then(() => {
    updateTableStats();
    if (tableData.value.length > 0) {
      currentStep.value = 1;
    }
  });
})
</script>

<template>
  <div class="backend-gen-container">
    <!-- 步骤条 -->
    <a-steps :current="currentStep" size="small" style="margin-bottom: 20px">
      <a-step title="导入 Schema"/>
      <a-step title="表配置"/>
      <a-step title="生成配置"/>
    </a-steps>

    <!-- ====== 步骤 0: 导入 Schema ====== -->
    <div v-if="currentStep === 0" class="step-content">
      <!-- 打开项目 - 空状态 -->
      <div v-if="!projectInfo && !projectError" class="project-empty-card" @click="handleOpenProject">
        <div class="project-empty-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#1890ff" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            <line x1="12" y1="11" x2="12" y2="17"/>
            <line x1="9" y1="14" x2="15" y2="14"/>
          </svg>
        </div>
        <div v-if="projectLoading" style="font-weight: 500; color: #1890ff">
          <a-spin size="small"/> 正在识别项目...
        </div>
        <template v-else>
          <div class="project-empty-title">点击打开项目目录</div>
          <div class="project-empty-desc">选择 Go 微服务项目的根目录</div>
        </template>
      </div>

      <!-- 打开项目 - 错误状态 -->
      <div v-else-if="projectError" class="project-error-card">
        <div class="project-error-left">
          <div class="project-error-indicator">
            <span class="project-error-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#ff4d4f" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
            </span>
            <span class="project-error-label">项目识别失败</span>
          </div>
          <div class="project-error-msg">{{ projectError }}</div>
          <div class="project-error-hint">请确保目录下包含 go.mod 文件</div>
        </div>
        <a-button size="small" type="primary" @click="handleOpenProject">重新选择</a-button>
      </div>

      <!-- 项目已打开 -->
      <div v-if="projectInfo" class="project-opened-card">
        <div class="project-opened-left">
          <div class="project-opened-indicator">
            <span class="project-opened-dot"></span>
            <span class="project-opened-label">项目已就绪</span>
          </div>
          <div class="project-opened-name">{{ projectInfo.ModPath }}</div>
          <div class="project-opened-meta">
            <span class="meta-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
              Go {{ projectInfo.GoVersion }}
            </span>
            <span class="meta-divider">|</span>
            <span class="meta-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
              {{ projectInfo.Services?.length ?? 0 }} 个服务
            </span>
            <span class="meta-divider">|</span>
            <span class="meta-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              {{ projectInfo.HasApi ? '已定义 API' : '未定义 API' }}
            </span>
          </div>
        </div>
        <a-button size="small" @click="handleOpenProject">切换项目</a-button>
      </div>

      <!-- 导入方式 -->
      <a-card title="导入 Schema" size="small">
        <a-radio-group v-model:value="importSource" style="margin-bottom: 16px">
          <a-radio-button value="database">数据库导入</a-radio-button>
          <a-radio-button value="file">本地文件</a-radio-button>
          <a-radio-button value="remote">远程地址</a-radio-button>
          <a-radio-button value="editor">SQL 编辑器</a-radio-button>
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
                <a-form-item label="数据库类型" name="dbType">
                  <a-select v-model:value="dbFormData.dbType">
                    <a-select-option v-for="db in dbTypes" :key="db.value" :value="db.value">
                      {{ db.label }}
                    </a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="16">
                <a-form-item label="数据源名称 (DSN)" name="dsn">
                  <a-textarea
                    v-model:value="dbFormData.dsn"
                    placeholder="示例: mysql://user:password@localhost:3306/dbname?charset=utf8mb4"
                    :rows="2"
                  />
                </a-form-item>
              </a-col>
            </a-row>
            <div style="display: flex; gap: 8px">
              <a-button @click="handleTestConnection" :loading="dbTestLoading">
                测试连接
              </a-button>
              <a-button type="primary" @click="handleDatabaseImport" :loading="dbLoading">
                导入表结构
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
              <div style="font-size: 32px; color: #1890ff; margin-bottom: 8px">&#128196;</div>
              <div v-if="fileLoading" style="font-weight: 500">
                <a-spin size="small"/> 正在导入...
              </div>
              <div v-else style="font-weight: 500; margin-bottom: 4px">
                {{ selectedFileName || '点击或拖拽 SQL 文件到此处' }}
              </div>
              <div style="color: #999; font-size: 12px">支持 .sql / .ddl 格式的 DDL 文件</div>
            </div>
          </div>
        </div>

        <!-- 远程 URL -->
        <div v-if="importSource === 'remote'">
          <a-input-search
            v-model:value="remoteUrl"
            placeholder="输入 SQL DDL 文件 URL，如 https://example.com/schema.sql"
            enter-button="拉取"
            :loading="remoteLoading"
            @search="handleFetchRemote"
            style="margin-bottom: 12px"
          />
          <a-alert v-if="!remoteUrl" message="请输入可公开访问的 SQL DDL 文件地址" type="info" show-icon/>
        </div>

        <!-- SQL 编辑器 -->
        <div v-if="importSource === 'editor'">
          <a-textarea
            v-model:value="sqlContent"
            placeholder="请粘贴或输入 SQL DDL 语句（CREATE TABLE ...）..."
            :auto-size="{ minRows: 10, maxRows: 20 }"
            style="font-family: 'Courier New', monospace; font-size: 12px; margin-bottom: 12px;"
          />
          <div style="display: flex; gap: 8px">
            <a-button type="primary" @click="handleSqlImport" :disabled="!sqlContent.trim()">
              导入 SQL
            </a-button>
            <a-button @click="handleOpenSqlEditor">
              打开高级编辑器
            </a-button>
          </div>
        </div>

        <!-- 已导入提示 -->
        <div v-if="tableData.length > 0" style="margin-top: 16px; padding-top: 12px; border-top: 1px solid #f0f0f0;">
          <a-tag color="success">已导入 {{ tableData.length }} 张表</a-tag>
          <a-button type="link" size="small" @click="refreshTableData" style="margin-left: 8px">刷新</a-button>
        </div>
      </a-card>

      <div style="text-align: right; margin-top: 16px">
        <a-button type="primary" @click="handleNextFromImport" :disabled="tableData.length === 0">
          下一步：配置表
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 1: 表配置 ====== -->
    <div v-if="currentStep === 1" class="step-content">
      <a-card size="small">
        <template #title>
          <span>表配置 ({{ tableData.length }} 张表，{{ excludedCount }} 张已排除)</span>
        </template>
        <template #extra>
          <a-space>
            <a-button size="small" @click="openDatabaseImporter = true">追加导入</a-button>
            <a-button size="small" @click="openSqlImporter = true">SQL 导入</a-button>
          </a-space>
        </template>

        <vxe-table
          :data="tableData"
          :row-config="{ keyField: 'id' }"
          size="small"
          class="table-content"
        >
          <vxe-column field="tableName" title="表名" min-width="200"/>
          <vxe-column field="service" title="所属服务" min-width="180">
            <template #header>
              <div class="service-header">
                <span>所属服务</span>
                <a-select
                  v-model:value="quickSelectService"
                  :options="serviceOptions"
                  placeholder="一键全选"
                  style="width: 150px; margin-left: 8px"
                  @change="handleQuickSelectService"
                  allow-clear
                />
              </div>
            </template>
            <template #default="{ row }">
              <a-select
                v-model:value="row.service"
                :options="serviceOptions"
                placeholder="选择服务"
                style="width: 100%"
                @change="handleServiceChange(row)"
              />
            </template>
          </vxe-column>
          <vxe-column field="exclude" title="排除" width="80" align="center">
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

      <div style="display: flex; justify-content: space-between; margin-top: 16px">
        <a-button @click="currentStep = 0">上一步</a-button>
        <a-button type="primary" @click="currentStep = 2">
          下一步：生成配置
        </a-button>
      </div>
    </div>

    <!-- ====== 步骤 2: 生成配置 ====== -->
    <div v-if="currentStep === 2" class="step-content">
      <!-- 生成目标 -->
      <a-card title="生成目标" size="small" style="margin-bottom: 16px">
        <div style="display: flex; flex-direction: column; gap: 16px">
          <!-- gRPC -->
          <div class="target-card" :class="{ active: generateConfig.generateGrpc }">
            <div class="target-header">
              <a-checkbox v-model:checked="generateConfig.generateGrpc">
                <span class="target-title">gRPC 微服务</span>
              </a-checkbox>
              <a-tag color="blue" size="small">Go + Protobuf</a-tag>
            </div>
            <div v-if="generateConfig.generateGrpc" class="target-body">
              <a-form layout="inline">
                <a-form-item label="ORM 类型">
                  <a-select v-model:value="generateConfig.ormType" style="width: 120px">
                    <a-select-option v-for="item in ormTypes" :key="item.value" :value="item.value">
                      {{ item.label }}
                    </a-select-option>
                  </a-select>
                </a-form-item>
              </a-form>
            </div>
          </div>

          <!-- BFF -->
          <div class="target-card" :class="{ active: generateConfig.generateBff }">
            <div class="target-header">
              <a-checkbox v-model:checked="generateConfig.generateBff">
                <span class="target-title">BFF 微服务</span>
              </a-checkbox>
              <a-tag color="green" size="small">REST API</a-tag>
            </div>
            <div v-if="generateConfig.generateBff" class="target-body">
              <a-form layout="inline">
                <a-form-item label="BFF 服务名">
                  <a-input v-model:value="generateConfig.bffServiceName" style="width: 180px" placeholder="如 admin"/>
                </a-form-item>
              </a-form>
            </div>
          </div>
        </div>
      </a-card>

      <!-- 生成概览 -->
      <a-card title="生成概览" size="small">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="项目">{{ projectInfo?.ModPath || '-' }}</a-descriptions-item>
          <a-descriptions-item label="有效表数">{{ tableData.length - excludedCount }} / {{ tableData.length }}</a-descriptions-item>
          <a-descriptions-item label="生成 gRPC">{{ generateConfig.generateGrpc ? generateConfig.ormType + ' ORM' : '否' }}</a-descriptions-item>
          <a-descriptions-item label="生成 BFF">{{ generateConfig.generateBff ? generateConfig.bffServiceName : '否' }}</a-descriptions-item>
        </a-descriptions>
      </a-card>

      <div style="display: flex; justify-content: space-between; margin-top: 16px">
        <a-button @click="currentStep = 1">上一步</a-button>
        <a-button
          type="primary"
          danger
          :loading="confirmLoading"
          :disabled="!generateConfig.generateGrpc && !generateConfig.generateBff"
          @click="handleGenerate"
        >
          开始生成代码
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

/* 项目已打开 - 成功卡片 */
.project-opened-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: linear-gradient(135deg, #f6ffed 0%, #e8f5e9 100%);
  border: 1px solid #b7eb8f;
  border-radius: 10px;
  padding: 16px 20px;
  margin-bottom: 16px;
}

.project-opened-left {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.project-opened-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.project-opened-dot {
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

.project-opened-label {
  font-size: 13px;
  font-weight: 600;
  color: #389e0d;
}

.project-opened-name {
  font-size: 16px;
  font-weight: 700;
  color: #1a1a1a;
  font-family: 'Consolas', 'Courier New', monospace;
}

.project-opened-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #595959;
  font-size: 12px;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.meta-divider {
  color: #d9d9d9;
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
</style>

<script setup lang="ts">
import {ref, reactive} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  FolderOpenOutlined,
  ReloadOutlined,
  CloudUploadOutlined,
  SendOutlined,
} from '@ant-design/icons-vue'
import {
  OpenProject,
  SelectFolder,
  GetProjectInfo,
  GetRemoteConfigTypes,
  GetConfigServices,
  ExportConfigToRemote,
  ExportOneServiceConfig,
} from '../../../wailsjs/go/main/App'

const {t} = useI18n()

// ==================== 项目信息 ====================
const projectInfo = ref<any>()
const projectLoading = ref(false)

async function handleOpenProject() {
  try {
    const path = await SelectFolder()
    if (!path) return

    projectLoading.value = true
    const pi = await OpenProject(path)
    if (!pi || !pi.ModPath) {
      message.error(t('remoteConfig.project.noProject'))
      projectInfo.value = undefined
      return
    }
    projectInfo.value = pi
    message.success(t('common.success'))
    await loadServices()
  } catch (err) {
    message.error(t('remoteConfig.project.noProject'))
  } finally {
    projectLoading.value = false
  }
}

// ==================== 配置中心类型 ====================
const configTypes = ref<any[]>([])

async function loadConfigTypes() {
  try {
    const types = await GetRemoteConfigTypes()
    configTypes.value = types || []
  } catch (e) {
    console.error('加载配置中心类型失败:', e)
  }
}

// ==================== 远程配置表单 ====================
const remoteConfig = reactive({
  type: 'consul',
  endpoint: '',
  projectName: '',
  group: '',
  env: '',
  namespaceId: '',
})

function handleTypeChange() {
  // 根据类型设置默认端口
  switch (remoteConfig.type) {
    case 'consul':
      if (!remoteConfig.endpoint) remoteConfig.endpoint = '127.0.0.1:8500'
      break
    case 'etcd':
      if (!remoteConfig.endpoint) remoteConfig.endpoint = '127.0.0.1:2379'
      break
    case 'nacos':
      if (!remoteConfig.endpoint) remoteConfig.endpoint = '127.0.0.1:8848'
      break
  }
}

// ==================== 服务列表 ====================
interface ServiceItem {
  name: string
  configFiles: string[]
  configFolder: string
}

const services = ref<ServiceItem[]>([])
const servicesLoading = ref(false)
const selectedServiceNames = ref<string[]>([])

async function loadServices() {
  if (!projectInfo.value) return

  servicesLoading.value = true
  try {
    const list = await GetConfigServices()
    services.value = list || []
    selectedServiceNames.value = services.value.map(s => s.name)
    if (services.value.length > 0) {
      message.success(t('remoteConfig.service.loadSuccess', {count: services.value.length}))
    }
  } catch (e) {
    message.error(t('remoteConfig.service.loadFailed'))
  } finally {
    servicesLoading.value = false
  }
}

// ==================== 导出操作 ====================
const exporting = ref(false)
const exportingService = ref('')

async function handleExportAll() {
  if (!projectInfo.value) {
    message.warning(t('remoteConfig.project.openFirst'))
    return
  }

  if (services.value.length === 0) {
    message.warning(t('remoteConfig.export.noServices'))
    return
  }

  exporting.value = true
  try {
    const result = await ExportConfigToRemote(remoteConfig as any)
    if (result.success) {
      message.success(t('remoteConfig.export.success'))
    } else {
      message.error(t('remoteConfig.export.failed', {msg: result.error}))
    }
  } catch (e) {
    message.error(t('remoteConfig.export.failed', {msg: e}))
  } finally {
    exporting.value = false
  }
}

async function handleExportOne(serviceName: string) {
  if (!projectInfo.value) {
    message.warning(t('remoteConfig.project.openFirst'))
    return
  }

  exportingService.value = serviceName
  try {
    const result = await ExportOneServiceConfig(remoteConfig as any, serviceName)
    if (result.success) {
      message.success(t('remoteConfig.export.success') + ` (${serviceName})`)
    } else {
      message.error(t('remoteConfig.export.failed', {msg: result.error}))
    }
  } catch (e) {
    message.error(t('remoteConfig.export.failed', {msg: e}))
  } finally {
    exportingService.value = ''
  }
}

// ==================== 文件名提取 ====================
function getFileName(path: string): string {
  const parts = path.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || path
}

// 初始化
loadConfigTypes()
</script>

<template>
  <div class="remote-config-page">
    <!-- 项目选择 -->
    <div class="project-bar">
      <a-button type="primary" :loading="projectLoading" @click="handleOpenProject">
        <FolderOpenOutlined style="margin-right: 4px"/> {{ projectInfo ? t('remoteConfig.project.switchProject') : t('remoteConfig.project.selectProject') }}
      </a-button>
      <span v-if="projectInfo" class="project-info">
        {{ projectInfo.ModPath }}
      </span>
      <span v-else class="project-info project-info--empty">{{ t('remoteConfig.project.noProject') }}</span>
    </div>

    <div class="main-content">
      <!-- 左侧: 配置表单 -->
      <div class="config-panel">
        <a-card :title="t('remoteConfig.config.title')" size="small">
          <a-form layout="vertical">
            <a-form-item :label="t('remoteConfig.config.type')">
              <a-select v-model:value="remoteConfig.type" @change="handleTypeChange">
                <a-select-option v-for="ct in configTypes" :key="ct.value" :value="ct.value">
                  {{ ct.label }}
                </a-select-option>
              </a-select>
            </a-form-item>

            <a-form-item :label="t('remoteConfig.config.endpoint')" required>
              <a-input
                  v-model:value="remoteConfig.endpoint"
                  :placeholder="t('remoteConfig.config.endpointPlaceholder')"
              />
            </a-form-item>

            <a-form-item :label="t('remoteConfig.config.projectName')" required>
              <a-input
                  v-model:value="remoteConfig.projectName"
                  :placeholder="t('remoteConfig.config.projectNamePlaceholder')"
              />
            </a-form-item>

            <template v-if="remoteConfig.type === 'nacos'">
              <a-form-item :label="t('remoteConfig.config.group')">
                <a-input
                    v-model:value="remoteConfig.group"
                    :placeholder="t('remoteConfig.config.groupPlaceholder')"
                />
              </a-form-item>
              <a-form-item :label="t('remoteConfig.config.env')">
                <a-input
                    v-model:value="remoteConfig.env"
                    :placeholder="t('remoteConfig.config.envPlaceholder')"
                />
              </a-form-item>
              <a-form-item :label="t('remoteConfig.config.namespaceId')">
                <a-input
                    v-model:value="remoteConfig.namespaceId"
                    :placeholder="t('remoteConfig.config.namespaceIdPlaceholder')"
                />
              </a-form-item>
            </template>
          </a-form>
        </a-card>
      </div>

      <!-- 右侧: 服务列表 -->
      <div class="service-panel">
        <a-card size="small">
          <template #title>
            <div class="card-title-row">
              <span>{{ t('remoteConfig.service.title') }}</span>
              <div class="card-title-actions">
                <a-button size="small" :loading="servicesLoading" @click="loadServices" :disabled="!projectInfo">
                  <ReloadOutlined style="margin-right: 4px"/> {{ t('common.refresh') }}
                </a-button>
                <a-button
                    type="primary"
                    size="small"
                    :loading="exporting"
                    :disabled="services.length === 0"
                    @click="handleExportAll"
                >
                  <CloudUploadOutlined style="margin-right: 4px"/> {{ t('remoteConfig.export.exportAll') }}
                </a-button>
              </div>
            </div>
          </template>

          <a-spin :spinning="servicesLoading">
            <div v-if="services.length > 0" class="service-list">
              <div v-for="svc in services" :key="svc.name" class="service-item">
                <div class="service-header">
                  <div class="service-name">
                    <a-tag color="blue">{{ svc.name }}</a-tag>
                    <span class="file-count">{{ t('remoteConfig.service.fileCount', {count: svc.configFiles.length}) }}</span>
                  </div>
                  <a-button
                      size="small"
                      type="primary"
                      :loading="exportingService === svc.name"
                      @click="handleExportOne(svc.name)"
                  >
                    <SendOutlined style="margin-right: 4px"/> {{ t('remoteConfig.export.exportOne') }}
                  </a-button>
                </div>
                <div class="service-files">
                  <a-tag v-for="file in svc.configFiles" :key="file" color="default">
                    {{ getFileName(file) }}
                  </a-tag>
                </div>
              </div>
            </div>
            <a-empty v-else :description="t('remoteConfig.service.noServices')"/>
          </a-spin>
        </a-card>
      </div>
    </div>
  </div>
</template>

<style scoped>
.remote-config-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 12px;
}

.project-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.project-info {
  font-size: 13px;
  color: #595959;
}

.project-info--empty {
  color: #bfbfbf;
}

.main-content {
  flex: 1;
  display: flex;
  gap: 16px;
  overflow: hidden;
  min-height: 0;
}

.config-panel {
  width: 360px;
  flex-shrink: 0;
  overflow-y: auto;
}

.service-panel {
  flex: 1;
  overflow-y: auto;
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.card-title-actions {
  display: flex;
  gap: 8px;
}

.service-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.service-item {
  padding: 10px 12px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #f0f0f0;
}

.service-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.service-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-count {
  font-size: 12px;
  color: #8c8c8c;
}

.service-files {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>

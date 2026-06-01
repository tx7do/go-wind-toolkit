<script setup lang="ts">
import {ref, reactive, onMounted} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  FolderOpenOutlined,
  FolderOutlined,
  PlusOutlined,
  AppstoreAddOutlined,
} from '@ant-design/icons-vue'
import {
  SelectFolder, OpenProject, GetProjectInfo,
  GetDevServices, CreateProject, AddService,
} from '../../../wailsjs/go/main/App'

const {t} = useI18n()

const projectInfo = ref<any>(null)
const services = ref<any[]>([])
const activeSection = ref('overview')

// ==================== 打开项目 ====================
async function handleOpenProject() {
  try {
    const path = await SelectFolder()
    if (!path) return
    const pi = await OpenProject(path)
    if (!pi || !pi.ModPath) {
      message.error(t('projectManager.overview.noProject'))
      return
    }
    projectInfo.value = pi
    await loadServices()
    message.success(t('backend.project.ready'))
  } catch (err) {
    message.error(t('backend.project.openFailed'))
  }
}

async function loadServices() {
  try {
    const list = await GetDevServices()
    services.value = list || []
  } catch (e) {
    services.value = []
  }
}

// ==================== 创建项目 ====================
const createForm = reactive({
  name: '',
  module: '',
  repoUrl: '',
  branch: '',
  parentDir: '',
})
const creating = ref(false)

async function handleSelectParentDir() {
  try {
    const folder = await SelectFolder()
    if (folder) createForm.parentDir = folder
  } catch (e) {
    console.error(e)
  }
}

async function handleCreateProject() {
  if (!createForm.name.trim()) {
    message.warning(t('projectManager.create.name') + ' required')
    return
  }
  if (!createForm.parentDir) {
    message.warning(t('projectManager.create.parentDir') + ' required')
    return
  }
  creating.value = true
  try {
    const result = await CreateProject({
      name: createForm.name,
      module: createForm.module || createForm.name,
      repoUrl: createForm.repoUrl,
      branch: createForm.branch,
      parentDir: createForm.parentDir,
    })
    if (result.success) {
      message.success(t('projectManager.create.success'))
      // 自动打开新创建的项目
      const newPath = createForm.parentDir + '\\' + createForm.name
      const pi = await OpenProject(newPath)
      if (pi) {
        projectInfo.value = pi
        await loadServices()
      }
      createForm.name = ''
      createForm.module = ''
      createForm.repoUrl = ''
      createForm.branch = ''
      createForm.parentDir = ''
    } else {
      message.error(result.error || t('projectManager.create.failed'))
    }
  } catch (e) {
    message.error(t('projectManager.create.failed'))
  } finally {
    creating.value = false
  }
}

// ==================== 添加服务 ====================
const addForm = reactive({
  serviceName: '',
  servers: ['grpc'] as string[],
  dbClients: ['ent'] as string[],
})
const adding = ref(false)

const serverOptions = [
  {label: 'gRPC', value: 'grpc'},
  {label: 'REST/BFF', value: 'rest'},
]
const dbClientOptions = [
  {label: 'Ent', value: 'ent'},
  {label: 'GORM', value: 'gorm'},
  {label: 'Redis', value: 'redis'},
]

async function handleAddService() {
  if (!addForm.serviceName.trim()) {
    message.warning(t('projectManager.addService.serviceName') + ' required')
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
      message.success(t('projectManager.addService.success'))
      addForm.serviceName = ''
      await loadServices()
    } else {
      message.error(result.error || t('projectManager.addService.failed'))
    }
  } catch (e) {
    message.error(t('projectManager.addService.failed'))
  } finally {
    adding.value = false
  }
}

// 初始化：尝试加载已打开的项目
onMounted(async () => {
  try {
    const pi = await GetProjectInfo()
    if (pi && pi.ModPath) {
      projectInfo.value = pi
      await loadServices()
    }
  } catch (e) {
    // ignore
  }
})
</script>

<template>
  <div class="project-manager-page">
    <!-- 顶部项目操作栏 -->
    <div class="top-bar">
      <a-button type="primary" @click="handleOpenProject">
        <FolderOpenOutlined style="margin-right: 4px"/> {{ projectInfo ? t('backend.project.switchProject') : t('backend.project.clickToOpen') }}
      </a-button>
      <span v-if="projectInfo" class="project-path">
        {{ projectInfo.ModPath }}
      </span>
      <span v-else class="project-path project-path--empty">
        {{ t('projectManager.overview.noProject') }}
      </span>
    </div>

    <!-- 子Tab切换 -->
    <a-tabs v-model:activeKey="activeSection" size="small" style="margin-top: 12px">
      <!-- 项目概览 -->
      <a-tab-pane key="overview" :tab="t('projectManager.overview.title')">
        <div v-if="projectInfo" class="overview-section">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item :label="t('projectManager.overview.modPath')">
              {{ projectInfo.ModPath }}
            </a-descriptions-item>
            <a-descriptions-item :label="t('projectManager.overview.goVersion')">
              {{ projectInfo.GoVersion }}
            </a-descriptions-item>
          </a-descriptions>

          <a-card size="small" style="margin-top: 12px">
            <template #title>
              <span>{{ t('projectManager.overview.services') }}
                <a-tag color="blue">{{ t('projectManager.overview.serviceCount', {count: services.length}) }}</a-tag>
              </span>
            </template>
            <a-table v-if="services.length > 0" :data-source="services" :pagination="false" size="small" row-key="name">
              <a-table-column dataIndex="name" :title="t('projectManager.addService.serviceName')"/>
              <a-table-column dataIndex="hasServer" :title="t('projectManager.overview.hasServer')" width="80">
                <template #default="{record}">
                  <a-tag v-if="record.hasServer" color="green">OK</a-tag>
                  <a-tag v-else>-</a-tag>
                </template>
              </a-table-column>
              <a-table-column dataIndex="hasConfig" :title="t('projectManager.overview.hasConfig')" width="80">
                <template #default="{record}">
                  <a-tag v-if="record.hasConfig" color="green">OK</a-tag>
                  <a-tag v-else>-</a-tag>
                </template>
              </a-table-column>
              <a-table-column dataIndex="hasEnt" :title="t('projectManager.overview.hasEnt')" width="80">
                <template #default="{record}">
                  <a-tag v-if="record.hasEnt" color="blue">{{ (record.entSchemas || []).length }} schemas</a-tag>
                  <a-tag v-else>-</a-tag>
                </template>
              </a-table-column>
            </a-table>
            <a-empty v-else :description="t('devTools.service.noServices')" style="padding: 20px 0"/>
          </a-card>
        </div>
        <a-empty v-else :description="t('projectManager.overview.noProject')" style="padding: 40px 0"/>
      </a-tab-pane>

      <!-- 创建新项目 -->
      <a-tab-pane key="create" :tab="t('projectManager.create.title')">
        <a-card :title="t('projectManager.create.title')" size="small" style="max-width: 600px">
          <a-form layout="vertical">
            <a-form-item :label="t('projectManager.create.parentDir')" required>
              <a-input-group compact>
                <a-input v-model:value="createForm.parentDir" :placeholder="t('projectManager.create.parentDirPlaceholder')" style="width: calc(100% - 100px)" read-only/>
                <a-button type="primary" @click="handleSelectParentDir"><FolderOutlined style="margin-right: 4px"/> {{ t('projectManager.create.selectDir') }}</a-button>
              </a-input-group>
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('projectManager.create.name')" required>
                  <a-input v-model:value="createForm.name" :placeholder="t('projectManager.create.namePlaceholder')"/>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('projectManager.create.module')">
                  <a-input v-model:value="createForm.module" :placeholder="t('projectManager.create.modulePlaceholder')"/>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('projectManager.create.repoUrl')">
                  <a-input v-model:value="createForm.repoUrl" :placeholder="t('projectManager.create.repoPlaceholder')"/>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('projectManager.create.branch')">
                  <a-input v-model:value="createForm.branch" :placeholder="t('projectManager.create.branchPlaceholder')"/>
                </a-form-item>
              </a-col>
            </a-row>
            <a-button type="primary" :loading="creating" @click="handleCreateProject" block>
              <PlusOutlined style="margin-right: 4px"/> {{ t('projectManager.create.title') }}
            </a-button>
          </a-form>
        </a-card>
      </a-tab-pane>

      <!-- 添加服务 -->
      <a-tab-pane key="addService" :tab="t('projectManager.addService.title')">
        <a-card :title="t('projectManager.addService.title')" size="small" style="max-width: 600px">
          <a-form layout="vertical">
            <a-form-item :label="t('projectManager.addService.serviceName')" required>
              <a-input v-model:value="addForm.serviceName" :placeholder="t('projectManager.addService.serviceNamePlaceholder')"/>
            </a-form-item>
            <a-form-item :label="t('projectManager.addService.servers')">
              <a-checkbox-group v-model:value="addForm.servers">
                <a-checkbox v-for="opt in serverOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-checkbox>
              </a-checkbox-group>
            </a-form-item>
            <a-form-item :label="t('projectManager.addService.dbClients')">
              <a-checkbox-group v-model:value="addForm.dbClients">
                <a-checkbox v-for="opt in dbClientOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-checkbox>
              </a-checkbox-group>
            </a-form-item>
            <a-button type="primary" :loading="adding" :disabled="!projectInfo" @click="handleAddService" block>
              <AppstoreAddOutlined style="margin-right: 4px"/> {{ t('projectManager.addService.addBtn') }}
            </a-button>
            <p v-if="!projectInfo" style="color: #faad14; text-align: center; margin-top: 8px; font-size: 12px">
              {{ t('projectManager.overview.noProject') }}
            </p>
          </a-form>
        </a-card>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<style scoped>
.project-manager-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}
.project-path {
  font-size: 13px;
  color: #595959;
}
.project-path--empty {
  color: #bfbfbf;
}
.overview-section {
  padding: 4px 0;
}
</style>

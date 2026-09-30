<script setup lang="ts">
import {ref, reactive} from 'vue'
import {message, Modal} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  PlusOutlined,
  GlobalOutlined,
  FolderOpenOutlined,
} from '@ant-design/icons-vue'
import {CreateProject, SelectFolder} from '../../wailsjs/go/main/App'
import {useProject} from '../stores/project'

const {t} = useI18n()

const {
  projectInfo,
  projectError,
  projectLoading,
  hasProject,
  openProject,
  selectAndOpenProject,
} = useProject()

const emit = defineEmits<{
  (e: 'switchLocale'): void
  (e: 'projectOpened'): void
}>()

// ==================== 新建项目 ====================
const createVisible = ref(false)
const creating = ref(false)
const createForm = reactive({
  name: '',
  module: '',
  repoUrl: '',
  branch: '',
  parentDir: '',
})

async function handleSelectParentDir() {
  try {
    const folder = await SelectFolder()
    if (folder) createForm.parentDir = folder
  } catch (e) { /* ignore */ }
}

function showCreateModal() {
  createForm.name = ''
  createForm.module = ''
  createForm.repoUrl = ''
  createForm.branch = ''
  createForm.parentDir = ''
  createVisible.value = true
}

// 已打开项目时新建等于换项目：先讲清楚旧项目的表配置/DSN 会被丢弃。
function openCreateModal() {
  if (!hasProject.value) {
    showCreateModal()
    return
  }
  Modal.confirm({
    title: t('devTools.create.switchTitle'),
    content: t('devTools.create.switchContent'),
    okText: t('common.confirm'),
    cancelText: t('common.cancel'),
    onOk: showCreateModal,
  })
}

async function handleCreateProject() {
  if (!createForm.name.trim()) {
    message.warning(t('devTools.create.nameRequired'))
    return
  }
  if (!createForm.parentDir) {
    message.warning(t('devTools.create.dirRequired'))
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
      message.success(t('devTools.create.success'))
      createVisible.value = false
      if (!result.dir) {
        message.error(t('devTools.create.failed'))
        return
      }
      const opened = await openProject(result.dir)
      if (opened) emit('projectOpened')
    } else {
      message.error(result.error || t('devTools.create.failed'))
    }
  } catch (e) {
    message.error(t('devTools.create.failed'))
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="header">
    <div class="header-left">
      <span class="app-title">{{ t('app.title') }}</span>
      <a-divider type="vertical" class="project-divider"/>
      <template v-if="hasProject">
        <span class="project-name" :title="projectInfo?.ModPath">{{ projectInfo?.ModPath }}</span>
        <a-tag color="green">{{ t('backend.project.services', {count: projectInfo?.Services?.length ?? 0}) }}</a-tag>
        <a-tag v-if="projectInfo?.HasApi" color="blue">{{ t('backend.project.apiDefined') }}</a-tag>
        <span class="project-switch" @click="selectAndOpenProject">{{ t('backend.project.switchProject') }}</span>
      </template>
      <template v-else>
        <a-button size="small" type="primary" :loading="projectLoading" @click="selectAndOpenProject">
          <FolderOpenOutlined style="margin-right: 4px"/> {{ t('app.openProject') }}
        </a-button>
        <span class="project-error" :title="projectError">{{ projectError || t('app.noProject') }}</span>
      </template>
    </div>
    <div class="header-right">
      <a-button size="small" :type="hasProject ? 'default' : 'primary'" :ghost="!hasProject" @click="openCreateModal">
        <PlusOutlined style="margin-right: 4px"/> {{ hasProject ? t('devTools.create.btnSwitch') : t('devTools.create.btn') }}
      </a-button>
      <span class="lang-switch" @click="emit('switchLocale')"><GlobalOutlined style="margin-right: 4px"/> {{ t('header.switchLang') }}</span>
    </div>
  </div>

  <!-- 新建后端项目 Modal -->
  <a-modal v-model:open="createVisible" :title="t('devTools.create.title')" :confirm-loading="creating" @ok="handleCreateProject" :width="520">
    <a-form layout="vertical" style="margin-top: 16px">
      <a-form-item :label="t('devTools.create.parentDir')" required>
        <a-input-group compact>
          <a-input v-model:value="createForm.parentDir" :placeholder="t('devTools.create.dirPlaceholder')" style="width: calc(100% - 100px)" read-only/>
          <a-button type="primary" @click="handleSelectParentDir">{{ t('devTools.create.selectDir') }}</a-button>
        </a-input-group>
      </a-form-item>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item :label="t('devTools.create.name')" required>
            <a-input v-model:value="createForm.name" :placeholder="t('devTools.create.namePlaceholder')"/>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item :label="t('devTools.create.module')">
            <a-input v-model:value="createForm.module" :placeholder="t('devTools.create.modulePlaceholder')"/>
          </a-form-item>
        </a-col>
      </a-row>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item :label="t('devTools.create.repoUrl')">
            <a-input v-model:value="createForm.repoUrl" :placeholder="t('devTools.create.repoPlaceholder')"/>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item :label="t('devTools.create.branch')">
            <a-input v-model:value="createForm.branch" :placeholder="t('devTools.create.branchPlaceholder')"/>
          </a-form-item>
        </a-col>
      </a-row>
    </a-form>
  </a-modal>
</template>

<style scoped>
.header {
  height: 48px;
  width: 100%;
  max-width: 100vw;
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-title {
  font-size: 15px;
  font-weight: 600;
  color: #262626;
}

.project-divider {
  margin: 0 4px;
  border-color: #e0e0e0;
}

.project-name {
  max-width: 40vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: #262626;
}

.project-switch {
  font-size: 12px;
  color: #1890ff;
  cursor: pointer;
  white-space: nowrap;
}

.project-switch:hover {
  text-decoration: underline;
}

.project-error {
  max-width: 30vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #ff4d4f;
}

.lang-switch {
  font-size: 13px;
  color: #595959;
  cursor: pointer;
  padding: 4px 12px;
  border-radius: 4px;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
  user-select: none;
}

.lang-switch:hover {
  color: #1890ff;
  border-color: #91caff;
  background: #f0f7ff;
}
</style>

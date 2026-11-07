<script setup lang="ts">
import {computed} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {EyeOutlined, CheckCircleOutlined, CloseCircleOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'
import {useProject} from '../../../stores/project'
import {detectLanguage} from '../../../utils/language'
import MonacoEditor from '../../backend/MonacoEditor.vue'

const {t} = useI18n()
const {projectInfo} = useProject()
const {
  ormType, partitions, hasPartitions,
  backendPlan, planningLoading, applyingLoading,
  backendWithTests, setBackendWithTests,
  previewFile, previewContent, previewDrawerOpen,
  planBackend, applyBackend, discardBackendPlan, openPlanFile,
} = useAI()

// 计划待确认徽标
const planBadge = computed(() => {
  if (!backendPlan.value) return null
  return t('ai.badge.planPending', {
    newCount: backendPlan.value.newCount || 0,
    overwriteCount: backendPlan.value.overwriteCount || 0,
  })
})

// 预览阶段:主按钮是"预览变更";预览就绪后进入确认面板
async function handlePlan() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }
  if (!hasPartitions.value) {
    message.warning(t('ai.backend.noPartitions'))
    return
  }
  const result = await planBackend()
  if (!result?.success) {
    message.error(result?.error || t('ai.backend.planFailed'))
  }
}

async function handleApply() {
  const err = await applyBackend()
  if (err === '') {
    message.success(t('ai.backend.applySuccess'))
  } else {
    message.error(err || t('ai.backend.applyFailed'))
  }
}

async function handleDiscard() {
  await discardBackendPlan()
}

async function handleOpenFile(path: string) {
  const result = await openPlanFile(path)
  if (!result?.success) {
    message.error(result?.error || t('ai.backend.planExpired'))
  }
}
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span class="title-with-badge">
          {{ t('ai.backend.title') }}
          <a-tag v-if="planBadge" color="orange" class="status-badge">{{ planBadge }}</a-tag>
        </span>
      </div>
    </template>
    <a-form layout="vertical">
      <a-form-item :label="t('ai.backend.ormType')">
        <a-radio-group v-model:value="ormType" :disabled="!!backendPlan">
          <a-radio-button value="ent">Ent</a-radio-button>
          <a-radio-button value="gorm">GORM</a-radio-button>
        </a-radio-group>
      </a-form-item>

      <div class="partition-summary" v-if="hasPartitions">
        <h4>{{ t('ai.backend.partitionSummary') }}</h4>
        <div class="partition-list-compact">
          <div v-for="(p, idx) in partitions" :key="idx" class="partition-item-compact">
            <span class="partition-name">{{ p.serviceName }}</span>
            <span class="partition-table-count">{{ p.tables.length }} {{ t('ai.backend.tables') }}</span>
          </div>
        </div>
      </div>

      <div class="tests-row">
        <a-switch size="small" :checked="backendWithTests" @change="(v: any) => setBackendWithTests(!!v)"/>
        <span class="tests-label">{{ t('ai.backend.withTests') }}</span>
        <span class="tests-hint">{{ t('ai.backend.withTestsHint') }}</span>
      </div>

      <a-button
          type="primary"
          size="large"
          :loading="planningLoading"
          :disabled="!projectInfo || !hasPartitions || !!backendPlan"
          @click="handlePlan"
          block
      >
        <EyeOutlined style="margin-right: 4px"/> {{ t('ai.backend.planBtn') }}
      </a-button>
      <p v-if="!projectInfo" class="warning-text">{{ t('ai.backend.noProject') }}</p>
    </a-form>

    <!-- 确认面板:预览就绪后展示 -->
    <div v-if="backendPlan" class="plan-panel">
      <a-divider/>

      <div class="plan-summary">
        <a-tag color="green">{{ t('ai.backend.newFiles', {count: backendPlan.newCount || 0}) }}</a-tag>
        <a-tag color="orange">{{ t('ai.backend.overwriteFiles', {count: backendPlan.overwriteCount || 0}) }}</a-tag>
        <span class="plan-hint">{{ t('ai.backend.planHint') }}</span>
      </div>

      <a-collapse ghost size="small" class="plan-files">
        <a-collapse-panel :header="`${t('ai.backend.fileList')} (${backendPlan.files?.length || 0})`">
          <div class="plan-file-list">
            <div
                v-for="f in backendPlan.files"
                :key="f.path"
                class="plan-file-item"
                @click="handleOpenFile(f.path)"
            >
              <a-tag :color="f.isNew ? 'green' : 'orange'" class="file-state-tag">
                {{ f.isNew ? t('ai.backend.tagNew') : t('ai.backend.tagOverwrite') }}
              </a-tag>
              <span class="file-path">{{ f.path }}</span>
            </div>
          </div>
        </a-collapse-panel>
      </a-collapse>

      <a-alert type="info" show-icon style="margin-top: 8px">
        <template #message>{{ t('ai.backend.postProcessNote') }}</template>
        <template #description>
          <span class="postprocess-steps">{{ (backendPlan.postProcess || []).join(' → ') }}</span>
        </template>
      </a-alert>

      <div class="plan-actions">
        <a-button @click="handleDiscard" :disabled="applyingLoading">
          <CloseCircleOutlined style="margin-right: 4px"/> {{ t('ai.backend.cancelPlan') }}
        </a-button>
        <a-button type="primary" danger :loading="applyingLoading" @click="handleApply">
          <CheckCircleOutlined style="margin-right: 4px"/> {{ t('ai.backend.confirmWrite') }}
        </a-button>
      </div>
    </div>

    <!-- 单文件内容预览 -->
    <a-drawer
        v-model:open="previewDrawerOpen"
        :title="previewFile"
        width="640"
        placement="right"
    >
      <MonacoEditor
          :model-value="previewContent"
          :language="detectLanguage(previewFile)"
          :read-only="true"
          height="100%"
      />
    </a-drawer>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.title-with-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-badge {
  margin: 0;
  font-weight: 400;
}

.partition-summary {
  margin-bottom: 16px;
  padding: 12px;
  background: #f6f8fa;
  border-radius: 6px;
}

.partition-summary h4 {
  margin-bottom: 8px;
  font-size: 14px;
}

.partition-list-compact {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.partition-item-compact {
  display: flex;
  align-items: center;
  gap: 8px;
}

.partition-name {
  font-weight: 600;
  color: #1890ff;
  font-size: 14px;
}

.partition-table-count {
  font-size: 12px;
  color: #8c8c8c;
}

.warning-text {
  font-size: 12px;
  color: #faad14;
  margin-top: 8px;
  text-align: center;
}

.plan-panel {
  margin-top: 4px;
}

.plan-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.plan-hint {
  font-size: 12px;
  color: #8c8c8c;
}

.plan-file-list {
  max-height: 240px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.plan-file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.plan-file-item:hover {
  background: #f5f5f5;
}

.file-state-tag {
  margin: 0;
  flex-shrink: 0;
  font-size: 11px;
  line-height: 18px;
}

.file-path {
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}

.postprocess-steps {
  font-family: monospace;
  font-size: 12px;
}

.plan-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.tests-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.tests-label {
  font-size: 13px;
}

.tests-hint {
  font-size: 12px;
  color: #8c8c8c;
}
</style>

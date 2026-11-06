<script setup lang="ts">
import {computed} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {EyeOutlined, SearchOutlined, FileTextOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'
import {useProject} from '../../../stores/project'
import MonacoEditor from '../../backend/MonacoEditor.vue'

const {t} = useI18n()
const {projectInfo} = useProject()
const {
  reviewScanFiles, reviewScanWarnings, reviewScanLoading, reviewSelected,
  reviewLoading, reviewResult, reviewReports, reviewViewName,
  scanForReview, runReview, viewReview,
} = useAI()

// 勾选项按路径展示,附文件大小
const fileOptions = computed(() =>
    reviewScanFiles.value.map(f => ({
      label: `${f.path} (${Math.ceil(f.size / 1024)}KB)`,
      value: f.path,
    }))
)

async function handleScan() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }
  const result = await scanForReview()
  if (!result?.success) {
    message.error(result?.error || t('ai.review.scanFailed'))
    return
  }
  message.success(t('ai.review.filesFound', {count: reviewScanFiles.value.length}))
}

async function handleReview() {
  if (reviewSelected.value.length === 0) {
    message.warning(t('ai.review.noFiles'))
    return
  }
  const result = await runReview()
  if (result?.success) {
    message.success(t('ai.review.success'))
  } else {
    message.error(result?.error || t('ai.review.failed'))
  }
}

async function handleViewReport(file: any) {
  const result = await viewReview(file)
  if (!result?.success) {
    message.error(result?.error || t('ai.review.failed'))
  }
}
</script>

<template>
  <a-card :title="t('ai.review.title')" size="small">
    <p class="step-desc">{{ t('ai.review.desc') }}</p>

    <div class="scan-bar">
      <a-button type="primary" :loading="reviewScanLoading" :disabled="!projectInfo" @click="handleScan">
        <SearchOutlined style="margin-right: 4px"/> {{ t('ai.review.scanBtn') }}
      </a-button>
      <span v-if="reviewScanFiles.length > 0" class="scan-summary">
        {{ t('ai.review.filesFound', {count: reviewScanFiles.length}) }}
      </span>
    </div>

    <a-alert
        v-if="reviewScanWarnings.length > 0"
        :message="reviewScanWarnings.join('; ')"
        type="warning"
        show-icon
        style="margin-top: 8px"
    />

    <!-- 文件勾选:默认全选,审查超大仓库时可勾掉生成物 -->
    <div v-if="fileOptions.length > 0" class="file-select">
      <h4>{{ t('ai.review.selectFiles') }}</h4>
      <a-checkbox-group v-model:value="reviewSelected" class="file-checkboxes">
        <a-checkbox v-for="opt in fileOptions" :key="opt.value" :value="opt.value" class="file-checkbox">
          {{ opt.label }}
        </a-checkbox>
      </a-checkbox-group>
      <a-button
          type="primary"
          :loading="reviewLoading"
          :disabled="reviewSelected.length === 0"
          style="margin-top: 12px"
          @click="handleReview"
      >
        <EyeOutlined style="margin-right: 4px"/> {{ t('ai.review.startBtn') }}
      </a-button>
    </div>

    <!-- 历史报告列表 -->
    <div v-if="reviewReports.length > 0" class="report-list">
      <h4>{{ t('ai.review.reports') }}</h4>
      <div
          v-for="r in reviewReports"
          :key="r.name"
          class="report-item"
          :class="{'report-item--active': reviewViewName === r.name}"
          @click="handleViewReport(r)"
      >
        <FileTextOutlined style="margin-right: 6px"/>
        <span class="report-name">{{ r.name }}</span>
        <span class="report-time">{{ r.modTime }}</span>
      </div>
    </div>

    <!-- 审查结果:实时流式输出或回看的历史报告 -->
    <div v-if="reviewResult" class="review-result">
      <a-divider/>
      <div v-if="reviewViewName" class="viewing-bar">
        <a-tag color="blue">{{ t('ai.review.viewing', {name: reviewViewName}) }}</a-tag>
      </div>
      <MonacoEditor v-model="reviewResult" language="markdown" height="400px" :read-only="true"/>
    </div>
  </a-card>
</template>

<style scoped>
.step-desc {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 12px;
}

.scan-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.scan-summary {
  font-size: 13px;
  color: #595959;
}

.file-select {
  margin-top: 12px;
}

.file-select h4 {
  font-size: 14px;
  margin-bottom: 8px;
}

.file-checkboxes {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 220px;
  overflow-y: auto;
  padding: 8px;
  background: #fafafa;
  border-radius: 6px;
}

.file-checkbox {
  margin: 0;
  word-break: break-all;
}

.report-list {
  margin-top: 16px;
}

.report-list h4 {
  font-size: 14px;
  margin-bottom: 8px;
}

.report-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
}

.report-item:hover {
  background: #f5f5f5;
}

.report-item--active {
  background: #e6f7ff;
}

.report-name {
  font-size: 13px;
}

.report-time {
  margin-left: auto;
  font-size: 12px;
  color: #8c8c8c;
}

.review-result {
  margin-top: 8px;
}

.viewing-bar {
  margin-bottom: 8px;
}
</style>

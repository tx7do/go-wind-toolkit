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
  reviewBatches, reviewBatchTotal, reviewIssues,
  scanForReview, runReview, viewReview, cancelTask,
} = useAI()

const reviewBadge = computed(() =>
    reviewReports.value.length > 0 ? t('ai.badge.reports', {count: reviewReports.value.length}) : null
)

// 严重度 → 标签颜色与文案
const severityMeta: Record<string, {color: string; key: string}> = {
  critical: {color: 'red', key: 'ai.review.sevCritical'},
  major: {color: 'orange', key: 'ai.review.sevMajor'},
  minor: {color: 'blue', key: 'ai.review.sevMinor'},
  info: {color: 'default', key: 'ai.review.sevInfo'},
}

function severityColor(sev: string): string {
  return severityMeta[sev]?.color || 'default'
}

function severityLabel(sev: string): string {
  return t(severityMeta[sev]?.key || 'ai.review.sevInfo')
}

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
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span class="title-with-badge">
          {{ t('ai.review.title') }}
          <a-tag v-if="reviewBadge" color="green" class="status-badge">{{ reviewBadge }}</a-tag>
        </span>
      </div>
    </template>
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
      <a-button v-if="reviewLoading" danger style="margin-top: 12px; margin-left: 8px" @click="cancelTask('review')">
        {{ t('ai.cancel') }}
      </a-button>

      <!-- map-reduce 批次进度:逐批点亮 -->
      <div v-if="reviewLoading && reviewBatchTotal > 0" class="batch-progress">
        <span class="batch-progress-title">{{ t('ai.review.batchProgress', {total: reviewBatchTotal}) }}</span>
        <a-tag
            v-for="b in reviewBatches"
            :key="b.index"
            :color="b.issueCount > 0 ? 'orange' : 'green'"
        >
          {{ b.index }}/{{ reviewBatchTotal }} · {{ b.label }} · {{ b.issueCount }}
        </a-tag>
      </div>
    </div>

    <!-- 结构化 issue 列表:按严重度着色 -->
    <div v-if="reviewIssues.length > 0" class="issue-list">
      <h4>{{ t('ai.review.issuesCount', {count: reviewIssues.length}) }}</h4>
      <div v-for="(iss, idx) in reviewIssues" :key="idx" class="issue-item">
        <a-tag :color="severityColor(iss.severity)" class="sev-tag">{{ severityLabel(iss.severity) }}</a-tag>
        <div class="issue-body">
          <div class="issue-title">{{ iss.title }}</div>
          <div class="issue-loc">{{ iss.file }}<template v-if="(iss.line ?? 0) > 0">:{{ iss.line }}</template></div>
          <div v-if="iss.detail" class="issue-detail">{{ iss.detail }}</div>
        </div>
      </div>
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

.batch-progress {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.batch-progress-title {
  font-size: 12px;
  color: #8c8c8c;
}

.issue-list {
  margin-top: 16px;
}

.issue-list h4 {
  font-size: 14px;
  margin-bottom: 8px;
}

.issue-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
}

.issue-item:hover {
  background: #fafafa;
}

.sev-tag {
  margin: 0;
  flex-shrink: 0;
}

.issue-body {
  flex: 1;
  min-width: 0;
}

.issue-title {
  font-size: 13px;
  font-weight: 600;
}

.issue-loc {
  font-family: monospace;
  font-size: 12px;
  color: #8c8c8c;
  word-break: break-all;
}

.issue-detail {
  font-size: 12px;
  color: #595959;
  margin-top: 2px;
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

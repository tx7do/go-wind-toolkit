<script setup lang="ts">
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {TableOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'

const {t} = useI18n()
const {ddlContent, partitions, partitionWarnings, partitionLoading, hasPartitions, partition} = useAI()

async function handlePartition() {
  if (!ddlContent.value.trim()) {
    message.warning(t('ai.partition.ddlRequired'))
    return
  }
  const result = await partition()
  if (result?.success) {
    message.success(t('ai.partition.success'))
  } else {
    message.error(result?.error || t('ai.partition.failed'))
  }
}
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span>{{ t('ai.partition.title') }}</span>
        <a-button type="primary" :loading="partitionLoading" @click="handlePartition">
          <TableOutlined style="margin-right: 4px"/> {{ t('ai.partition.generateBtn') }}
        </a-button>
      </div>
    </template>

    <!-- 幻觉表名剔除/跨服务去重等清洗告警 -->
    <a-alert
        v-if="partitionWarnings.length > 0"
        type="warning"
        show-icon
        style="margin-bottom: 8px"
    >
      <template #message>{{ t('ai.partition.warnings') }}</template>
      <template #description>
        <ul class="warning-list">
          <li v-for="(w, idx) in partitionWarnings" :key="idx">{{ w }}</li>
        </ul>
      </template>
    </a-alert>

    <div v-if="hasPartitions" class="partition-list">
      <a-card v-for="(p, idx) in partitions" :key="idx" size="small" class="partition-card">
        <template #title>
          <span class="partition-name">{{ p.serviceName }}</span>
        </template>
        <p class="partition-desc">{{ p.description }}</p>
        <div class="partition-tables">
          <a-tag v-for="table in p.tables" :key="table" color="blue">{{ table }}</a-tag>
        </div>
      </a-card>
    </div>
    <a-empty v-else :description="t('ai.partition.empty')"/>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.warning-list {
  margin: 0;
  padding-left: 18px;
}

.partition-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.partition-card {
  background: #fafafa;
}

.partition-name {
  font-weight: 600;
  color: #1890ff;
  font-size: 14px;
}

.partition-desc {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 8px;
}

.partition-tables {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>

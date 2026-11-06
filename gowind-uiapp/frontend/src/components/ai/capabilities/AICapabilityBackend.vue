<script setup lang="ts">
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {ThunderboltOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'
import {useProject} from '../../../stores/project'

const {t} = useI18n()
const {projectInfo} = useProject()
const {
  ormType, backendGenerating, partitions, hasPartitions, generateBackend,
} = useAI()

async function handleGenerate() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }
  if (!hasPartitions.value) {
    message.warning(t('ai.backend.noPartitions'))
    return
  }
  const err = await generateBackend()
  if (err === '') {
    message.success(t('ai.backend.generateSuccess'))
  } else {
    message.error(t('ai.backend.generateFailed', {msg: err}))
  }
}
</script>

<template>
  <a-card :title="t('ai.backend.title')" size="small">
    <a-form layout="vertical">
      <a-form-item :label="t('ai.backend.ormType')">
        <a-radio-group v-model:value="ormType">
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

      <a-button
          type="primary"
          size="large"
          :loading="backendGenerating"
          :disabled="!projectInfo || !hasPartitions"
          @click="handleGenerate"
          block
      >
        <ThunderboltOutlined style="margin-right: 4px"/> {{ t('ai.backend.generateBtn') }}
      </a-button>
      <p v-if="!projectInfo" class="warning-text">{{ t('ai.backend.noProject') }}</p>
    </a-form>
  </a-card>
</template>

<style scoped>
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
</style>

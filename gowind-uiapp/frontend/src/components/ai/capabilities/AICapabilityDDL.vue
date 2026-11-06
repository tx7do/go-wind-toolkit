<script setup lang="ts">
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {RobotOutlined, SaveOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'
import MonacoEditor from '../../backend/MonacoEditor.vue'

const {t} = useI18n()
const {
  requirements, ddlContent, ddlGenerating, ddlAttempt,
  generateDDL, saveArtifacts, cancelTask,
} = useAI()

async function handleGenerate() {
  if (!requirements.value.trim()) {
    message.warning(t('ai.requirements.required'))
    return
  }
  const result = await generateDDL()
  if (result?.success) {
    message.success(t('ai.ddl.generateSuccess'))
  } else {
    message.error(result?.error || t('ai.ddl.generateFailed'))
  }
}

async function handleSave() {
  const err = await saveArtifacts()
  if (err === '') {
    message.success(t('ai.artifact.saved'))
  } else {
    message.error(err)
  }
}
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span>{{ t('ai.ddl.title') }}</span>
        <div class="card-actions">
          <a-button :disabled="ddlGenerating || !ddlContent" @click="handleSave">
            <SaveOutlined style="margin-right: 4px"/> {{ t('ai.artifact.saveBtn') }}
          </a-button>
          <a-button v-if="ddlGenerating" danger @click="cancelTask('ddl')">
            {{ t('ai.cancel') }}
          </a-button>
          <a-button type="primary" :loading="ddlGenerating" @click="handleGenerate">
            <RobotOutlined style="margin-right: 4px"/> {{ t('ai.ddl.generateBtn') }}
          </a-button>
        </div>
      </div>
    </template>

    <a-form layout="vertical">
      <a-form-item :label="t('ai.requirements.label')">
        <a-textarea
            v-model:value="requirements"
            :placeholder="t('ai.requirements.placeholder')"
            :auto-size="{minRows: 6, maxRows: 14}"
            style="font-size: 14px"
        />
      </a-form-item>
    </a-form>

    <!-- attempt > 1 表示 DDL 未通过解析校验,后端正在带错误自动修复重试 -->
    <a-alert
        v-if="ddlGenerating && ddlAttempt > 1"
        :message="t('ai.ddl.retrying', {attempt: ddlAttempt})"
        type="warning"
        show-icon
        style="margin-bottom: 8px"
    />

    <MonacoEditor v-model="ddlContent" db-type="mysql" height="360px"/>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.card-actions {
  display: flex;
  gap: 8px;
}
</style>

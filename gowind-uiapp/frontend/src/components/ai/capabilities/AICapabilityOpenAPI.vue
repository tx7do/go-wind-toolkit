<script setup lang="ts">
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {SearchOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'
import {useProject} from '../../../stores/project'

const {t} = useI18n()
const {projectInfo} = useProject()
const {openapiFiles, openapiMessage, openapiLoading, findOpenAPI} = useAI()

async function handleFind() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }
  const result = await findOpenAPI()
  if (!result?.success) {
    message.error(result?.error || t('ai.openapi.searchFailed'))
    return
  }
  if (openapiFiles.value.length === 0) {
    message.info(openapiMessage.value || t('ai.openapi.notFound'))
  } else {
    message.success(t('ai.openapi.found', {count: openapiFiles.value.length}))
  }
}
</script>

<template>
  <a-card :title="t('ai.openapi.title')" size="small">
    <p class="step-desc">{{ t('ai.openapi.desc') }}</p>
    <a-button type="primary" :loading="openapiLoading" :disabled="!projectInfo" @click="handleFind">
      <SearchOutlined style="margin-right: 4px"/> {{ t('ai.openapi.searchBtn') }}
    </a-button>

    <div v-if="openapiFiles.length > 0" class="openapi-results">
      <h4>{{ t('ai.openapi.foundFiles') }}</h4>
      <a-list size="small" :data-source="openapiFiles">
        <template #renderItem="{item}">
          <a-list-item>
            <a-typography-text code>{{ item }}</a-typography-text>
          </a-list-item>
        </template>
      </a-list>
    </div>
  </a-card>
</template>

<style scoped>
.step-desc {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 12px;
}

.openapi-results {
  margin-top: 16px;
}

.openapi-results h4 {
  font-size: 14px;
  margin-bottom: 8px;
}
</style>

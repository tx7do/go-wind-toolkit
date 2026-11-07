<script setup lang="ts">
import {computed, reactive, ref} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {SaveOutlined, ApiOutlined} from '@ant-design/icons-vue'
import {GetAIConfig, SetAIConfig, GetAIProviderPresets, TestAIConnection} from '../../../../wailsjs/go/main/App'
import type {ai} from '../../../../wailsjs/go/models'
import {useAI} from '../../../stores/ai'

const {t} = useI18n()
const {aiConfigured, checkConfigured} = useAI()

// 标题栏就绪徽标:aiConfigured 三态,null 时不显示
const configBadge = computed(() => {
  if (aiConfigured.value === true) return {color: 'green', text: t('ai.config.badgeReady')}
  if (aiConfigured.value === false) return {color: 'orange', text: t('ai.config.badgeNotConfigured')}
  return null
})

interface AIConfigData {
  provider: string
  baseUrl: string
  apiKey: string
  azureApiVersion: string
  model: string
  temperature: number
  maxTokens: number
}

const aiConfig = reactive<AIConfigData>({
  provider: 'openai',
  baseUrl: 'https://api.openai.com/v1',
  apiKey: '',
  azureApiVersion: '',
  model: 'gpt-4o',
  temperature: 0.7,
  maxTokens: 4096,
})

const providerPresets = ref<any[]>([])
const testingConnection = ref(false)

async function loadAIConfig() {
  try {
    const config = await GetAIConfig()
    if (config) {
      Object.assign(aiConfig, config)
    }
    const presets = await GetAIProviderPresets()
    providerPresets.value = presets || []
  } catch (e) {
    console.error('加载AI配置失败:', e)
  }
}

function handleProviderChange() {
  const preset = providerPresets.value.find(p => p.value === aiConfig.provider)
  if (preset && preset.baseUrl) {
    aiConfig.baseUrl = preset.baseUrl
  }
}

async function handleSaveConfig() {
  try {
    await SetAIConfig(aiConfig as any)
    message.success(t('ai.config.saved'))
    checkConfigured()
  } catch (e) {
    message.error(t('ai.config.saveFailed'))
  }
}

async function handleTestConnection() {
  testingConnection.value = true
  try {
    await SetAIConfig(aiConfig as any)
    const result = await TestAIConnection()
    if (result.success) {
      message.success(t('ai.config.connectSuccess'))
    } else {
      message.error(result.error || t('ai.config.connectFailed'))
    }
    checkConfigured()
  } catch (e) {
    message.error(t('ai.config.connectFailed'))
  } finally {
    testingConnection.value = false
  }
}

loadAIConfig()
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span>{{ t('ai.config.title') }}</span>
        <a-tag v-if="configBadge" :color="configBadge.color" class="config-badge">{{ configBadge.text }}</a-tag>
      </div>
    </template>
    <a-form layout="vertical">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item :label="t('ai.config.provider')">
            <a-select v-model:value="aiConfig.provider" @change="handleProviderChange">
              <a-select-option v-for="p in providerPresets" :key="p.value" :value="p.value">
                {{ p.name }}
              </a-select-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item :label="t('ai.config.model')">
            <a-input v-model:value="aiConfig.model" :placeholder="t('ai.config.modelPlaceholder')"/>
          </a-form-item>
        </a-col>
      </a-row>
      <a-form-item :label="t('ai.config.baseUrl')">
        <a-input v-model:value="aiConfig.baseUrl" :placeholder="t('ai.config.baseUrlPlaceholder')"/>
      </a-form-item>
      <a-form-item v-if="aiConfig.provider === 'azure'" :label="t('ai.config.azureApiVersion')">
        <a-input v-model:value="aiConfig.azureApiVersion" :placeholder="t('ai.config.azureApiVersionPlaceholder')"/>
      </a-form-item>
      <a-form-item :label="t('ai.config.apiKey')">
        <a-input-password v-model:value="aiConfig.apiKey" :placeholder="t('ai.config.apiKeyPlaceholder')"/>
      </a-form-item>
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item :label="t('ai.config.temperature')">
            <a-slider v-model:value="aiConfig.temperature" :min="0" :max="2" :step="0.1"/>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item :label="t('ai.config.maxTokens')">
            <a-input-number v-model:value="aiConfig.maxTokens" :min="256" :max="128000" :step="256" style="width: 100%"/>
          </a-form-item>
        </a-col>
      </a-row>
      <div class="config-actions">
        <a-button type="primary" ghost @click="handleSaveConfig">
          <SaveOutlined style="margin-right: 4px"/> {{ t('ai.config.save') }}
        </a-button>
        <a-button type="primary" :loading="testingConnection" @click="handleTestConnection">
          <ApiOutlined style="margin-right: 4px"/> {{ t('ai.config.testConnection') }}
        </a-button>
      </div>
    </a-form>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.config-badge {
  margin: 0;
}

.config-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>

<script setup lang="ts">
import {ref, reactive, computed} from 'vue'
import {message, Modal} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {
  GetAIConfig,
  SetAIConfig,
  GetAIProviderPresets,
  TestAIConnection,
  AIGenerateDDL,
  AIPartitionMicroservices,
  AIGenerateBackendCode,
  AIFindOpenAPIFiles,
  AIReviewCode,
  OpenProject,
  SelectFolder,
  GetProjectInfo,
} from '../../../wailsjs/go/main/App'
import type {ai} from '../../../wailsjs/go/models'

import MonacoEditor from '../backend/MonacoEditor.vue'

const {t} = useI18n()

// ==================== 步骤控制 ====================
type AIStep = 'config' | 'requirements' | 'ddl' | 'partition' | 'backendGen' | 'openapi' | 'review'
const currentStep = ref<number>(0)
const stepKeys: AIStep[] = ['config', 'requirements', 'ddl', 'partition', 'backendGen', 'openapi', 'review']

const steps = computed(() => [
  {title: t('ai.steps.aiConfig')},
  {title: t('ai.steps.requirements')},
  {title: t('ai.steps.generateDDL')},
  {title: t('ai.steps.partition')},
  {title: t('ai.steps.backendGen')},
  {title: t('ai.steps.openapi')},
  {title: t('ai.steps.review')},
])

// ==================== 项目信息 ====================
const projectInfo = ref<any>()
const projectLoading = ref(false)

async function handleOpenProject() {
  try {
    const path = await SelectFolder()
    if (!path) return

    projectLoading.value = true
    const pi = await OpenProject(path)
    if (!pi || !pi.ModPath) {
      message.error(t('ai.project.noProject'))
      projectInfo.value = undefined
      return
    }
    projectInfo.value = pi
    message.success(t('ai.project.ready'))
  } catch (err) {
    message.error(t('ai.project.openFailed'))
  } finally {
    projectLoading.value = false
  }
}

// ==================== Step 1: AI 配置 ====================
interface AIConfigData {
  provider: string
  baseUrl: string
  apiKey: string
  model: string
  temperature: number
  maxTokens: number
}

const aiConfig = reactive<AIConfigData>({
  provider: 'openai',
  baseUrl: 'https://api.openai.com/v1',
  apiKey: '',
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

async function handleProviderChange() {
  const preset = providerPresets.value.find(p => p.value === aiConfig.provider)
  if (preset && preset.baseUrl) {
    aiConfig.baseUrl = preset.baseUrl
  }
}

async function handleSaveConfig() {
  try {
    await SetAIConfig(aiConfig as any)
    message.success(t('ai.config.saved'))
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
  } catch (e) {
    message.error(t('ai.config.connectFailed'))
  } finally {
    testingConnection.value = false
  }
}

// ==================== Step 2: 需求输入 ====================
const requirements = ref('')

// ==================== Step 3: DDL 生成 ====================
const ddlContent = ref('')
const ddlGenerating = ref(false)

async function handleGenerateDDL() {
  if (!requirements.value.trim()) {
    message.warning(t('ai.requirements.required'))
    return
  }

  ddlGenerating.value = true
  try {
    const result = await AIGenerateDDL(requirements.value)
    if (result.success) {
      ddlContent.value = result.content
      message.success(t('ai.ddl.generateSuccess'))
    } else {
      message.error(result.error || t('ai.ddl.generateFailed'))
    }
  } catch (e) {
    message.error(t('ai.ddl.generateFailed'))
  } finally {
    ddlGenerating.value = false
  }
}

// ==================== Step 4: 微服务划分 ====================
interface PartitionItem {
  serviceName: string
  tables: string[]
  description: string
}

const partitions = ref<PartitionItem[]>([])
const partitionLoading = ref(false)

async function handlePartition() {
  if (!ddlContent.value.trim()) {
    message.warning(t('ai.partition.ddlRequired'))
    return
  }

  partitionLoading.value = true
  try {
    const result = await AIPartitionMicroservices(ddlContent.value)
    if (result.success) {
      partitions.value = result.partitions || []
      message.success(t('ai.partition.success'))
    } else {
      message.error(result.error || t('ai.partition.failed'))
    }
  } catch (e) {
    message.error(t('ai.partition.failed'))
  } finally {
    partitionLoading.value = false
  }
}

const hasPartitions = computed(() => partitions.value.length > 0)

// ==================== Step 5: 后端代码生成 ====================
const ormType = ref('ent')
const backendGenerating = ref(false)

async function handleGenerateBackend() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }

  if (partitions.value.length === 0) {
    message.warning(t('ai.backend.noPartitions'))
    return
  }

  backendGenerating.value = true
  try {
    const err = await AIGenerateBackendCode(ddlContent.value, ormType.value, partitions.value as any)
    if (err && err !== '') {
      message.error(t('ai.backend.generateFailed', {msg: err}))
    } else {
      message.success(t('ai.backend.generateSuccess'))
    }
  } catch (e) {
    message.error(t('ai.backend.generateFailed', {msg: e}))
  } finally {
    backendGenerating.value = false
  }
}

// ==================== Step 6: OpenAPI 查找 ====================
const openapiFiles = ref<string[]>([])
const openapiLoading = ref(false)

async function handleFindOpenAPI() {
  if (!projectInfo.value) {
    message.warning(t('ai.project.openFirst'))
    return
  }

  openapiLoading.value = true
  try {
    const result = await AIFindOpenAPIFiles()
    if (result.success) {
      openapiFiles.value = result.files || []
      if (openapiFiles.value.length === 0) {
        message.info(result.message || t('ai.openapi.notFound'))
      } else {
        message.success(t('ai.openapi.found', {count: openapiFiles.value.length}))
      }
    } else {
      message.error(result.error || t('ai.openapi.searchFailed'))
    }
  } catch (e) {
    message.error(t('ai.openapi.searchFailed'))
  } finally {
    openapiLoading.value = false
  }
}

// ==================== Step 7: AI 代码审查 ====================
const reviewResult = ref('')
const reviewLoading = ref(false)
const reviewFileContents = reactive<Record<string, string>>({})

async function handleReview() {
  if (Object.keys(reviewFileContents).length === 0) {
    message.warning(t('ai.review.noFiles'))
    return
  }

  reviewLoading.value = true
  try {
    const result = await AIReviewCode(reviewFileContents)
    if (result.success) {
      reviewResult.value = result.content
      message.success(t('ai.review.success'))
    } else {
      message.error(result.error || t('ai.review.failed'))
    }
  } catch (e) {
    message.error(t('ai.review.failed'))
  } finally {
    reviewLoading.value = false
  }
}

// ==================== 步骤导航 ====================
function nextStep() {
  if (currentStep.value < steps.value.length - 1) {
    currentStep.value++
  }
}

function prevStep() {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

function canNext(): boolean {
  const key = stepKeys[currentStep.value]
  switch (key) {
    case 'config':
      return !!aiConfig.baseUrl && !!aiConfig.model
    case 'requirements':
      return !!requirements.value.trim()
    case 'ddl':
      return !!ddlContent.value.trim()
    case 'partition':
      return hasPartitions.value
    case 'backendGen':
    case 'openapi':
    case 'review':
      return true
    default:
      return false
  }
}

// 初始化
loadAIConfig()
</script>

<template>
  <div class="ai-assistant-page">
    <!-- 项目选择 -->
    <div class="project-bar">
      <a-button type="primary" :loading="projectLoading" @click="handleOpenProject">
        {{ projectInfo ? t('ai.project.switchProject') : t('ai.project.selectProject') }}
      </a-button>
      <span v-if="projectInfo" class="project-info">
        {{ projectInfo.ModPath }} ({{ projectInfo.Services?.length || 0 }} {{ t('ai.project.services') }})
      </span>
      <span v-else class="project-info project-info--empty">{{ t('ai.project.noProjectOpen') }}</span>
    </div>

    <!-- 步骤条 -->
    <a-steps :current="currentStep" size="small" class="steps-bar">
      <a-step v-for="(step, idx) in steps" :key="idx" :title="step.title"/>
    </a-steps>

    <!-- 步骤内容 -->
    <div class="step-content">
      <!-- Step 0: AI 配置 -->
      <div v-if="currentStep === 0" class="step-panel">
        <a-card :title="t('ai.config.title')" size="small">
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
                  <a-input-number v-model:value="aiConfig.maxTokens" :min="256" :max="128000" :step="256"
                                  style="width: 100%"/>
                </a-form-item>
              </a-col>
            </a-row>
            <div class="config-actions">
              <a-button @click="handleSaveConfig">{{ t('ai.config.save') }}</a-button>
              <a-button type="primary" :loading="testingConnection" @click="handleTestConnection">
                {{ t('ai.config.testConnection') }}
              </a-button>
            </div>
          </a-form>
        </a-card>
      </div>

      <!-- Step 1: 需求输入 -->
      <div v-if="currentStep === 1" class="step-panel">
        <a-card :title="t('ai.requirements.title')" size="small">
          <a-form layout="vertical">
            <a-form-item :label="t('ai.requirements.label')">
              <a-textarea
                  v-model:value="requirements"
                  :placeholder="t('ai.requirements.placeholder')"
                  :auto-size="{minRows: 10, maxRows: 25}"
                  style="font-size: 14px"
              />
            </a-form-item>
          </a-form>
        </a-card>
      </div>

      <!-- Step 2: DDL 生成 -->
      <div v-if="currentStep === 2" class="step-panel">
        <a-card size="small">
          <template #title>
            <div class="card-title-row">
              <span>{{ t('ai.ddl.title') }}</span>
              <a-button type="primary" :loading="ddlGenerating" @click="handleGenerateDDL">
                {{ t('ai.ddl.generateBtn') }}
              </a-button>
            </div>
          </template>
          <MonacoEditor v-model="ddlContent" db-type="mysql" height="400px"/>
        </a-card>
      </div>

      <!-- Step 3: 微服务划分 -->
      <div v-if="currentStep === 3" class="step-panel">
        <a-card size="small">
          <template #title>
            <div class="card-title-row">
              <span>{{ t('ai.partition.title') }}</span>
              <a-button type="primary" :loading="partitionLoading" @click="handlePartition">
                {{ t('ai.partition.generateBtn') }}
              </a-button>
            </div>
          </template>
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
      </div>

      <!-- Step 4: 后端代码生成 -->
      <div v-if="currentStep === 4" class="step-panel">
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
                :disabled="!projectInfo"
                @click="handleGenerateBackend"
                block
            >
              {{ t('ai.backend.generateBtn') }}
            </a-button>
            <p v-if="!projectInfo" class="warning-text">{{ t('ai.backend.noProject') }}</p>
          </a-form>
        </a-card>
      </div>

      <!-- Step 5: OpenAPI -->
      <div v-if="currentStep === 5" class="step-panel">
        <a-card :title="t('ai.openapi.title')" size="small">
          <p class="step-desc">{{ t('ai.openapi.desc') }}</p>
          <a-button type="primary" :loading="openapiLoading" :disabled="!projectInfo" @click="handleFindOpenAPI">
            {{ t('ai.openapi.searchBtn') }}
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
      </div>

      <!-- Step 6: AI 代码审查 -->
      <div v-if="currentStep === 6" class="step-panel">
        <a-card :title="t('ai.review.title')" size="small">
          <p class="step-desc">{{ t('ai.review.desc') }}</p>
          <a-button
              type="primary"
              :loading="reviewLoading"
              :disabled="!projectInfo"
              @click="handleReview"
          >
            {{ t('ai.review.startBtn') }}
          </a-button>

          <div v-if="reviewResult" class="review-result">
            <a-divider/>
            <MonacoEditor v-model="reviewResult" language="markdown" height="400px" :read-only="true"/>
          </div>
        </a-card>
      </div>
    </div>

    <!-- 底部导航 -->
    <div class="step-footer">
      <a-button v-if="currentStep > 0" @click="prevStep">
        {{ t('common.prevStep') }}
      </a-button>
      <div style="flex: 1"></div>
      <a-button v-if="currentStep < steps.length - 1" type="primary" :disabled="!canNext()" @click="nextStep">
        {{ t('common.nextStep') }}
      </a-button>
    </div>
  </div>
</template>

<style scoped>
.ai-assistant-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 12px;
}

.project-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}

.project-info {
  font-size: 13px;
  color: #595959;
}

.project-info--empty {
  color: #bfbfbf;
}

.steps-bar {
  flex-shrink: 0;
}

.step-content {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.step-panel {
  padding: 4px 0;
}

.config-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.partition-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
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

.partition-table-count {
  font-size: 12px;
  color: #8c8c8c;
}

.step-desc {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 12px;
}

.warning-text {
  font-size: 12px;
  color: #faad14;
  margin-top: 8px;
  text-align: center;
}

.openapi-results {
  margin-top: 16px;
}

.openapi-results h4 {
  font-size: 14px;
  margin-bottom: 8px;
}

.review-result {
  margin-top: 8px;
}

.step-footer {
  display: flex;
  align-items: center;
  padding-top: 8px;
  border-top: 1px solid #f0f0f0;
  flex-shrink: 0;
}
</style>

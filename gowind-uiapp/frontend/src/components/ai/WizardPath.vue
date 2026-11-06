<script setup lang="ts">
import {ref, computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {LeftOutlined, RightOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../stores/ai'
import AICapabilityDDL from './capabilities/AICapabilityDDL.vue'
import AICapabilityPartition from './capabilities/AICapabilityPartition.vue'
import AICapabilityBackend from './capabilities/AICapabilityBackend.vue'

// 新手向导:需求→DDL→划分→后端的固定链路,复用能力面板的同一批组件
// (产物真值在 stores/ai.ts,所以向导与面板随时可互切、互不丢状态)。
// OpenAPI 查找与代码审查不在产物链上,只在面板中出现。
const {t} = useI18n()
const {ddlContent, hasPartitions, mode} = useAI()

const currentStep = ref(0)
const stepCount = 3

const canNext = computed(() => {
  switch (currentStep.value) {
    case 0:
      return !!ddlContent.value.trim()
    case 1:
      return hasPartitions.value
    default:
      return true
  }
})

function nextStep() {
  if (currentStep.value < stepCount - 1 && canNext.value) {
    currentStep.value++
  }
}

function prevStep() {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}
</script>

<template>
  <div class="wizard-path">
    <div class="wizard-header">
      <span class="wizard-hint">{{ t('ai.wizard.hint') }}</span>
      <a-button link @click="mode = 'panel'">{{ t('ai.wizard.configHint') }}</a-button>
    </div>

    <a-steps :current="currentStep" size="small" class="steps-bar">
      <a-step :title="t('ai.steps.generateDDL')"/>
      <a-step :title="t('ai.steps.partition')"/>
      <a-step :title="t('ai.steps.backendGen')"/>
    </a-steps>

    <div class="step-content">
      <div v-show="currentStep === 0"><AICapabilityDDL/></div>
      <div v-show="currentStep === 1"><AICapabilityPartition/></div>
      <div v-show="currentStep === 2"><AICapabilityBackend/></div>
    </div>

    <div class="step-footer">
      <a-button v-if="currentStep > 0" @click="prevStep">
        <LeftOutlined style="margin-right: 4px"/> {{ t('common.prevStep') }}
      </a-button>
      <div style="flex: 1"></div>
      <a-button v-if="currentStep < stepCount - 1" type="primary" :disabled="!canNext" @click="nextStep">
        {{ t('common.nextStep') }} <RightOutlined style="margin-left: 4px"/>
      </a-button>
    </div>
  </div>
</template>

<style scoped>
.wizard-path {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 12px;
  min-height: 0;
}

.wizard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.wizard-hint {
  font-size: 13px;
  color: #8c8c8c;
}

.steps-bar {
  flex-shrink: 0;
}

.step-content {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.step-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  flex-shrink: 0;
}
</style>

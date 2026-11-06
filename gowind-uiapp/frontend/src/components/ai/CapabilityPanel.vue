<script setup lang="ts">
import {useI18n} from 'vue-i18n'
import AICapabilityConfig from './capabilities/AICapabilityConfig.vue'
import AICapabilityDDL from './capabilities/AICapabilityDDL.vue'
import AICapabilityPartition from './capabilities/AICapabilityPartition.vue'
import AICapabilityBackend from './capabilities/AICapabilityBackend.vue'
import AICapabilityOpenAPI from './capabilities/AICapabilityOpenAPI.vue'
import AICapabilityReview from './capabilities/AICapabilityReview.vue'

// 能力面板:六个能力各自独立、互不依赖触发顺序;
// 有产物依赖的能力(划分/后端)在使用处自行校验并提示。
// 产物自动保存到项目 .gowind/ai/ 目录,换项目/重开应用后自动恢复。
const {t} = useI18n()
</script>

<template>
  <div class="capability-panel">
    <a-row :gutter="[12, 12]">
      <a-col :span="12"><AICapabilityConfig/></a-col>
      <a-col :span="12"><AICapabilityDDL/></a-col>
      <a-col :span="12"><AICapabilityPartition/></a-col>
      <a-col :span="12"><AICapabilityBackend/></a-col>
      <a-col :span="12"><AICapabilityOpenAPI/></a-col>
      <a-col :span="12"><AICapabilityReview/></a-col>
    </a-row>
  </div>
</template>

<style scoped>
.capability-panel {
  overflow-y: auto;
  min-height: 0;
}

/* 卡片等高拉伸,避免两列高度错位 */
.capability-panel :deep(.ant-col) {
  display: flex;
}

.capability-panel :deep(.ant-col > *){
  width: 100%;
}
</style>

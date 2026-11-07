<script setup lang="ts">
import {nextTick, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'
import {FolderOpenOutlined, SettingOutlined} from '@ant-design/icons-vue'
import {useProject} from '../../stores/project'
import {useAI} from '../../stores/ai'
import CapabilityPanel from './CapabilityPanel.vue'
import WizardPath from './WizardPath.vue'

const {t} = useI18n()

// 项目信息(全局唯一真值,见 stores/project.ts)
const {projectInfo, hasProject, projectLoading, projectError, selectAndOpenProject} = useProject()

// AI 能力状态:流事件绑定、换项目重置与产物恢复都收敛在 store 的 initAI 里
const {mode, aiConfigured, initAI} = useAI()

onMounted(() => {
  initAI()
})

// 首次运行引导:切到面板,滚动到配置卡并高亮闪烁、聚焦 API Key 输入框。
// 配置卡是面板第一张卡,通常已在视区——滚动无位移,必须另有可见反馈,
// 否则按钮看起来"没有响应"。
async function goToConfig() {
  mode.value = 'panel'
  await nextTick()
  const card = document.getElementById('ai-config-card')
  card?.scrollIntoView({behavior: 'smooth', block: 'start'})
  if (card) {
    card.classList.remove('ai-config-flash')
    void (card as HTMLElement).offsetWidth // 强制重排,重启动画
    card.classList.add('ai-config-flash')
    window.setTimeout(() => card.classList.remove('ai-config-flash'), 1600)
  }
  const apiKeyInput = document.getElementById('ai-api-key-input') as HTMLInputElement | null
  apiKeyInput?.focus({preventScroll: true})
}
</script>

<template>
  <div class="ai-assistant-page">
    <!-- 首次运行引导:未配置 API Key 时所有生成类能力都不可用 -->
    <a-alert v-if="aiConfigured === false" type="warning" show-icon class="setup-banner">
      <template #message>{{ t('ai.banner.notConfigured') }}</template>
      <template #action>
        <a-button size="small" danger type="primary" @click="goToConfig">
          <SettingOutlined style="margin-right: 4px"/> {{ t('ai.banner.goConfig') }}
        </a-button>
      </template>
    </a-alert>

    <!-- 未打开项目时给出入口；已打开项目由顶栏全局展示，本页不重复 -->
    <div v-if="!hasProject" class="project-bar">
      <a-button type="primary" :loading="projectLoading" @click="selectAndOpenProject">
        <FolderOpenOutlined style="margin-right: 4px"/> {{ t('ai.project.selectProject') }}
      </a-button>
      <span class="project-info project-info--empty">{{ projectError || t('ai.project.noProjectOpen') }}</span>
    </div>

    <!-- 形态切换:能力面板(默认)或新手向导,两者共享同一份产物状态 -->
    <div class="mode-bar">
      <a-segmented
          v-model:value="mode"
          :options="[
            {label: t('ai.mode.panel'), value: 'panel'},
            {label: t('ai.mode.wizard'), value: 'wizard'},
          ]"
      />
      <span class="mode-hint">{{ mode === 'panel' ? t('ai.mode.panelHint') : t('ai.mode.wizardHint') }}</span>
    </div>

    <div class="mode-content">
      <CapabilityPanel v-if="mode === 'panel'"/>
      <WizardPath v-else/>
    </div>
  </div>
</template>

<style scoped>
.setup-banner {
  flex-shrink: 0;
}

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

.mode-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.mode-hint {
  font-size: 13px;
  color: #8c8c8c;
}

.mode-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>

<style>
/* goToConfig 的定位反馈:目标是 CapabilityPanel 渲染的 #ai-config-card(a-col),
   跨组件作用域,必须放全局块。卡片通常已在视区,滚动无位移,闪烁给出"已定位"反馈。 */
@keyframes aiConfigFlash {
  0%, 100% { box-shadow: 0 0 0 0 rgba(250, 173, 20, 0); }
  25% { box-shadow: 0 0 0 6px rgba(250, 173, 20, 0.45); }
  60% { box-shadow: 0 0 0 2px rgba(250, 173, 20, 0.25); }
}

.ai-config-flash {
  animation: aiConfigFlash 1.5s ease-in-out 1;
  border-radius: 8px;
}
</style>

<script setup lang="ts">
import {Tabs} from 'ant-design-vue';
import {useI18n} from 'vue-i18n'

import Header from "./Header.vue";
import CodeGeneratorPage from "./backend/CodeGeneratorPage.vue";
import FrontendCodeGenPage from "./frontend/FrontendCodeGenPage.vue";
import RemoteConfigPage from "./remote-config/RemoteConfigPage.vue";

const {t} = useI18n()

const emit = defineEmits<{
  (e: 'switchLocale'): void
}>()

const settingList = [
  {
    key: '1',
    nameKey: 'tabs.backend' as const,
    component: CodeGeneratorPage,
  },
  {
    key: '2',
    nameKey: 'tabs.frontend' as const,
    component: FrontendCodeGenPage,
  },
  {
    key: '3',
    nameKey: 'tabs.remoteConfig' as const,
    component: RemoteConfigPage,
  },
];
</script>

<template>
  <div class="page-container">
    <Header @switch-locale="emit('switchLocale')"/>
    <div class="layout-wrapper">
      <Tabs
          tab-position="left"
          class="tabs-container"
      >
        <template v-for="item in settingList" :key="item.key">
          <a-tab-pane :tab="t(item.nameKey)">
            <component :is="item.component"/>
          </a-tab-pane>
        </template>
      </Tabs>
    </div>
  </div>
</template>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100%;
  overflow: hidden;
  background-color: #ffffff;
}

.layout-wrapper {
  flex: 1;
  overflow: hidden;
  background-color: #ffffff;
}

.tabs-container {
  height: 100%;
  background-color: #ffffff;
}

:deep(.ant-tabs) {
  height: 100%;
  display: flex;
  flex-direction: row;
  background-color: #ffffff;
}

:deep(.ant-tabs-nav) {
  background-color: #f5f5f5;
  border-right: 1px solid #f0f0f0;
}

:deep(.ant-tabs-tabpane) {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 16px;
  box-sizing: border-box;
  overflow: auto;
  background-color: #ffffff;
}

:deep(.ant-tabs-content) {
  height: 100%;
}

:deep(.ant-tabs-content-holder) {
  flex: 1;
  overflow: hidden;
  background-color: #ffffff;
}
</style>

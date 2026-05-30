<script setup lang="ts">
import {useI18n} from 'vue-i18n'

const {t} = useI18n()

const emit = defineEmits<{
  (e: 'switchLocale'): void
}>()
</script>

<template>
  <div class="header">
    <div class="header-left">
      <span class="app-title">{{ t('app.title') }}</span>
    </div>
    <div class="header-right">
      <span class="lang-switch" @click="emit('switchLocale')">{{ t('header.switchLang') }}</span>
    </div>
  </div>
</template>

<style scoped>
.header {
  height: 48px;
  width: 100%;
  max-width: 100vw;
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-title {
  font-size: 15px;
  font-weight: 600;
  color: #262626;
}

.lang-switch {
  font-size: 13px;
  color: #595959;
  cursor: pointer;
  padding: 4px 12px;
  border-radius: 4px;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
  user-select: none;
}

.lang-switch:hover {
  color: #1890ff;
  border-color: #91caff;
  background: #f0f7ff;
}
</style>

<script lang="ts" setup>
import {ref, computed} from 'vue'
import Index from "./components/index.vue";
import i18n from "./i18n";

const currentLocale = computed(() => i18n.global.locale.value)

// Ant Design Vue 语言包
const antLocale = ref<any>(undefined)

async function loadAntLocale(lang: string) {
  if (lang === 'en-US') {
    const mod = await import('ant-design-vue/es/locale/en_US')
    antLocale.value = mod.default
  } else {
    const mod = await import('ant-design-vue/es/locale/zh_CN')
    antLocale.value = mod.default
  }
}

// 初始加载
loadAntLocale(currentLocale.value)

function switchLocale() {
  const newLocale = currentLocale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  i18n.global.locale.value = newLocale
  localStorage.setItem('locale', newLocale)
  loadAntLocale(newLocale)
}
</script>

<template>
  <a-config-provider
      :locale="antLocale"
      :theme="{
      components: {
        Radio: {
          colorPrimary: '#00b96b',
        },
      },
    }"
  >
    <Index @switch-locale="switchLocale"/>
  </a-config-provider>
</template>

<style>
</style>

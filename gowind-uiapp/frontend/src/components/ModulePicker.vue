<script setup lang="ts">
import {ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {EventsOn, EventsOff} from '../../wailsjs/runtime'
import {detect} from '../../wailsjs/go/models'
import {useProject} from '../stores/project'

type ModuleCandidate = detect.ModuleCandidate

const {t} = useI18n()
const {openProject} = useProject()

const visible = ref(false)
const candidates = ref<ModuleCandidate[]>([])
const selected = ref('')

function onModulesFound(data: ModuleCandidate[] | { data: ModuleCandidate[] }) {
  const list = Array.isArray(data) ? data : (data?.data ?? [])
  if (!list.length) return
  candidates.value = list
  selected.value = list[0].Dir
  visible.value = true
}

EventsOn('project-modules-found', onModulesFound)
window.addEventListener('beforeunload', () => EventsOff('project-modules-found'))

async function handleConfirm() {
  const dir = selected.value
  visible.value = false
  if (dir) await openProject(dir)
}
</script>

<template>
  <a-modal
      v-model:open="visible"
      :title="t('app.selectModuleTitle')"
      :ok-text="t('app.confirm')"
      :cancel-text="t('app.cancel')"
      @ok="handleConfirm"
  >
    <p class="picker-hint">{{ t('app.selectModuleHint') }}</p>
    <a-radio-group v-model:value="selected" class="picker-group">
      <label
          v-for="c in candidates"
          :key="c.Dir"
          class="picker-item"
      >
        <a-radio :value="c.Dir">
          <span class="picker-mod">{{ c.ModPath }}</span>
          <span class="picker-rel">{{ c.RelPath }}</span>
        </a-radio>
      </label>
    </a-radio-group>
  </a-modal>
</template>

<style scoped>
.picker-hint {
  margin-bottom: 12px;
  color: rgba(0, 0, 0, 0.55);
}
.picker-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.picker-item {
  display: block;
}
.picker-mod {
  font-weight: 500;
}
.picker-rel {
  margin-left: 8px;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}
</style>

import {computed, ref} from 'vue'
import i18n from '../i18n'
import {EventsOn} from '../../wailsjs/runtime'
import {GetProjectInfo, OpenProject, SelectFolder} from '../../wailsjs/go/main/App'
import {detect} from '../../wailsjs/go/models'

const projectInfo = ref<detect.ProjectInfo | null>(null)
const projectError = ref('')
const projectLoading = ref(false)

const hasProject = computed(() => !!projectInfo.value?.ModPath)

function t(key: string): string {
  return i18n.global.t(key) as string
}

function errorMessage(err: unknown, fallback: string): string {
  const msg = err instanceof Error ? err.message : String(err ?? '')
  return msg.trim() || fallback
}

/**
 * 打开指定目录。返回 true 表示项目已就绪；
 * false 表示打开失败，或正等待模块选择器弹窗确认（choose）。
 */
async function openProject(path: string): Promise<boolean> {
  const res = await OpenProject(path)

  // choose: 模块选择器接管弹窗，保持当前项目状态不变。
  if (!res || res.Status === 'choose') return false

  if (res.Status !== 'opened' || !res.Project?.ModPath) {
    projectInfo.value = null
    projectError.value = t('app.projectInvalid')
    return false
  }

  projectInfo.value = res.Project
  projectError.value = ''
  return true
}

/** 弹出系统目录选择框并打开所选项目；用户取消时返回 false。 */
async function selectAndOpenProject(): Promise<boolean> {
  let path: string
  try {
    path = await SelectFolder()
  } catch {
    return false
  }
  if (!path) return false

  projectLoading.value = true
  projectError.value = ''
  try {
    return await openProject(path)
  } catch (err) {
    projectInfo.value = null
    projectError.value = errorMessage(err, t('app.projectOpenFailed'))
    return false
  } finally {
    projectLoading.value = false
  }
}

let bound = false

/**
 * 全局项目状态的唯一入口。首个调用方（根组件）负责挂上事件监听，
 * 之后所有页面共享同一份 projectInfo，不再各自缓存。
 */
export function useProject() {
  if (!bound) {
    bound = true

    EventsOn('project-opened', (pi: detect.ProjectInfo) => {
      if (pi?.ModPath) {
        projectInfo.value = pi
        projectError.value = ''
      }
    })

    EventsOn('config-cleaned', () => {
      projectInfo.value = null
      projectError.value = ''
    })

    GetProjectInfo().then(pi => {
      if (pi?.ModPath && !projectInfo.value) projectInfo.value = pi
    }).catch(() => { /* 启动时后端尚未就绪，等事件补齐 */ })
  }

  return {
    projectInfo,
    projectError,
    projectLoading,
    hasProject,
    openProject,
    selectAndOpenProject,
  }
}

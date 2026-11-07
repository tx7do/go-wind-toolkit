<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {RobotOutlined, SaveOutlined, SafetyCertificateOutlined} from '@ant-design/icons-vue'
import {AIValidateDDL} from '../../../../wailsjs/go/main/App'
import {useAI} from '../../../stores/ai'
import MonacoEditor from '../../backend/MonacoEditor.vue'

// 与 MonacoEditor 的 diagnostics prop 同形(script setup 不允许导出类型)
interface EditorDiagnostic {
  line: number
  column: number
  message: string
}

const {t} = useI18n()
const {
  requirements, ddlContent, ddlGenerating, ddlAttempt,
  generateDDL, saveArtifacts, cancelTask,
} = useAI()

// 标题徽标:DDL 就绪时显示行数
const ddlBadge = computed(() => {
  const lines = ddlContent.value.split('\n').filter(l => l.trim()).length
  return lines > 0 ? t('ai.badge.ddlReady', {lines}) : null
})

// 编辑器语法诊断:校验按钮/生成失败时回填,内容一变即清(陈旧标记误导)
const ddlDiagnostics = ref<EditorDiagnostic[]>([])
watch(ddlContent, () => {
  ddlDiagnostics.value = []
})

async function handleGenerate() {
  if (!requirements.value.trim()) {
    message.warning(t('ai.requirements.required'))
    return
  }
  const result = await generateDDL()
  if (result?.success) {
    message.success(t('ai.ddl.generateSuccess'))
  } else {
    message.error(result?.error || t('ai.ddl.generateFailed'))
    // 校验耗尽重试的失败消息带行列诊断,回填编辑器
    const diag = extractDiagnosticFromMessage(result?.error || '')
    if (diag) {
      ddlDiagnostics.value = [diag]
    }
  }
}

// 从「DDL 语法错误(第 N 行): line X column Y near ...」格式的错误串还原诊断
function extractDiagnosticFromMessage(msg: string): EditorDiagnostic | null {
  const m = msg.match(/line (\d+) column (\d+)/)
  if (!m) return null
  return {line: parseInt(m[1], 10), column: parseInt(m[2], 10), message: msg}
}

async function handleValidate() {
  if (!ddlContent.value.trim()) {
    message.warning(t('ai.partition.ddlRequired'))
    return
  }
  const result = await AIValidateDDL(ddlContent.value)
  if (result.success) {
    ddlDiagnostics.value = []
    message.success(t('ai.ddl.validOk'))
  } else {
    ddlDiagnostics.value = result.diagnostics || []
    message.warning(t('ai.ddl.validFailed', {count: ddlDiagnostics.value.length}))
  }
}

async function handleSave() {
  const err = await saveArtifacts()
  if (err === '') {
    message.success(t('ai.artifact.saved'))
  } else {
    message.error(err)
  }
}
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span class="title-with-badge">
          {{ t('ai.ddl.title') }}
          <a-tag v-if="ddlBadge && !ddlGenerating" color="green" class="status-badge">{{ ddlBadge }}</a-tag>
        </span>
        <div class="card-actions">
          <a-button :disabled="ddlGenerating || !ddlContent" @click="handleValidate">
            <SafetyCertificateOutlined style="margin-right: 4px"/> {{ t('ai.ddl.validateBtn') }}
          </a-button>
          <a-button :disabled="ddlGenerating || !ddlContent" @click="handleSave">
            <SaveOutlined style="margin-right: 4px"/> {{ t('ai.artifact.saveBtn') }}
          </a-button>
          <a-button v-if="ddlGenerating" danger @click="cancelTask('ddl')">
            {{ t('ai.cancel') }}
          </a-button>
          <a-button type="primary" :loading="ddlGenerating" @click="handleGenerate">
            <RobotOutlined style="margin-right: 4px"/> {{ t('ai.ddl.generateBtn') }}
          </a-button>
        </div>
      </div>
    </template>

    <a-form layout="vertical">
      <a-form-item :label="t('ai.requirements.label')">
        <a-textarea
            v-model:value="requirements"
            :placeholder="t('ai.requirements.placeholder')"
            :auto-size="{minRows: 6, maxRows: 14}"
            style="font-size: 14px"
        />
      </a-form-item>
    </a-form>

    <!-- attempt > 1 表示 DDL 未通过解析校验,后端正在带错误自动修复重试 -->
    <a-alert
        v-if="ddlGenerating && ddlAttempt > 1"
        :message="t('ai.ddl.retrying', {attempt: ddlAttempt})"
        type="warning"
        show-icon
        style="margin-bottom: 8px"
    />

    <MonacoEditor v-model="ddlContent" db-type="mysql" height="360px" :diagnostics="ddlDiagnostics"/>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.title-with-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-badge {
  margin: 0;
  font-weight: 400;
}

.card-actions {
  display: flex;
  gap: 8px;
}
</style>

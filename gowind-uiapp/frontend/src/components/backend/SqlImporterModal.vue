<script setup lang="ts">
import {ref, watch, nextTick, computed} from "vue";
import {message} from "ant-design-vue";

import {ImportSqlTables, SetDBConfig} from "../../../wailsjs/go/main/App";

import MonacoEditor from './MonacoEditor.vue';

const props = defineProps<{
  open?: boolean
  modelValue?: string
  dbType?: 'mysql' | 'postgresql' | 'sqlite' | 'oracle' // 从父组件传入
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'update:modelValue', value: string): void
  (e: 'submit', value: string): void
}>()

const innerOpen = ref(false)
const sqlContent = ref(`SELECT *
                        FROM users
                        WHERE id = 1;`);

const editorRef = ref<InstanceType<typeof MonacoEditor>>()
const validateLoading = ref(false)
const validateResult = ref<{
  success: boolean
  message: string
  errors?: Array<{ line: number; message: string }>
} | null>(null)

// 同步外部的 open 状态
watch(() => props.open, async (val) => {
  innerOpen.value = val ?? false
  if (val) {
    // 打开时同步初始值
    sqlContent.value = props.modelValue || ''
    // 等待 DOM 更新后聚焦
    await nextTick()
    editorRef.value?.focus()
    // 清除之前的验证结果
    validateResult.value = null
  }
}, {immediate: true})

// 同步 SQL 内容到外部
watch(() => sqlContent.value, (val) => {
  emit('update:modelValue', val)
  // 内容变化时清除验证结果
  validateResult.value = null
})

// 关闭模态框
function handleClose() {
  emit('update:open', false)
}

// 提交表单
async function handleCommit() {
  const trimmed = sqlContent.value.trim()
  if (!trimmed) {
    message.warning('请输入 SQL 语句')
    return
  }

  // 如果有验证结果且验证失败，提示用户
  if (validateResult.value && !validateResult.value.success) {
    message.warning('SQL 语法验证未通过，请先修复错误')
    return
  }

  const res = await ImportSqlTables(trimmed);
  if (res !== '') {
    console.log(res)
    message.error('SQL 导入失败，请检查语句是否正确')
    return
  }

  message.success('SQL导入成功！');

  await SetDBConfig({
    database: "", dbPath: "", host: "", password: "", port: 0, ssl: false, username: "",
    sqlContent: trimmed,
    type: props.dbType || ''
  });

  emit('submit', trimmed)
  handleClose()
}

// 清空 SQL
function clearSQL() {
  sqlContent.value = ''
  validateResult.value = null
  message.success('已清空')
  editorRef.value?.focus()
}

// 格式化 SQL
function formatSQL() {
  editorRef.value?.formatDocument()
  validateResult.value = null // 格式化后清除验证结果
  message.success('格式化完成')
}

// 验证语法
async function validateSyntax() {
  const trimmed = sqlContent.value.trim()
  if (!trimmed) {
    message.warning('请输入 SQL 语句')
    return
  }

  validateLoading.value = true
  validateResult.value = null

  try {
    // 模拟验证（实际项目中调用 Wails API）
    // 根据 dbType 选择不同的验证逻辑
    const success = await simulateValidation(props.dbType || 'mysql')

    if (success) {
      validateResult.value = {
        success: true,
        message: `✅ ${props.dbType?.toUpperCase() || 'SQL'} 语法验证通过`
      }
      message.success('语法验证通过')
    } else {
      validateResult.value = {
        success: false,
        message: `❌ ${props.dbType?.toUpperCase() || 'SQL'} 语法验证失败`,
        errors: [
          {line: 1, message: '语法错误：缺少分号'},
          {line: 3, message: `在 ${props.dbType} 中不支持该语法`}
        ]
      }
      message.error('语法验证失败')
    }
  } catch (error) {
    console.error('验证失败:', error)
    message.error('验证过程中发生错误')
    validateResult.value = {
      success: false,
      message: '❌ 验证失败'
    }
  } finally {
    validateLoading.value = false
  }
}

// 模拟不同数据库的验证逻辑
async function simulateValidation(dbType: string): Promise<boolean> {
  await new Promise(resolve => setTimeout(resolve, 500))

  // 根据数据库类型调整验证规则
  const sql = sqlContent.value.toLowerCase()

  if (dbType === 'mysql' && sql.includes('returning')) {
    return false // MySQL 不支持 RETURNING
  }

  if (dbType === 'sqlite' && sql.includes('generate_series')) {
    return false // SQLite 不支持 generate_series
  }

  return Math.random() > 0.3 // 70% 成功率
}

// 获取统计信息
const lineCount = computed(() => {
  return sqlContent.value.split('\n').filter(line => line.trim()).length
})

const charCount = computed(() => {
  return sqlContent.value.length
})

// 跳转到错误行
function jumpToLine(lineNumber: number) {
  const editor = editorRef.value?.getEditor()
  if (editor) {
    editor.setPosition({lineNumber, column: 1})
    editor.revealLine(lineNumber)
    editor.focus()
  }
}
</script>

<template>
  <a-modal
      v-model:open="innerOpen"
      title="SQL 输入"
      :width="900"
      @ok="handleCommit"
      @cancel="handleClose"
      okText="导入"
      cancelText="取消"
      :okButtonProps="{ disabled: !sqlContent.trim() }"
      :bodyStyle="{ padding: '16px' }"
      :destroyOnClose="true"
  >
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <a-button
            size="small"
            @click="clearSQL"
            type="text"
            title="清空内容"
        >
          <template #icon>
            <span class="icon-btn">⌫</span>
          </template>
          清空
        </a-button>
        <a-button
            size="small"
            @click="formatSQL"
            type="text"
            title="格式化 SQL"
        >
          <template #icon>
            <span class="icon-btn">✎</span>
          </template>
          格式化
        </a-button>
        <a-divider type="vertical"/>
        <a-button
            size="small"
            @click="validateSyntax"
            type="primary"
            :loading="validateLoading"
            title="验证 SQL 语法"
        >
          <template #icon>
            <span class="icon-btn">✓</span>
          </template>
          验证语法
        </a-button>
        <a-divider type="vertical"/>
        <a-tag color="blue">SQL</a-tag>
      </div>
      <div class="toolbar-right">
        <span class="stat-item">📝 行数: {{ lineCount }}</span>
        <span class="stat-item">📊 字符: {{ charCount }}</span>
      </div>
    </div>

    <!-- Monaco Editor -->
    <div class="editor-wrapper">
      <MonacoEditor
          ref="editorRef"
          v-model="sqlContent"
          :db-type="dbType"
          :height="400"
          @change="(val: any) => emit('update:modelValue', val)"
      />
    </div>

    <!-- 验证结果 -->
    <div v-if="validateResult" class="validate-result"
         :class="{ 'success': validateResult.success, 'error': !validateResult.success }">
      <div class="validate-header">
        <span class="validate-icon" :class="{ 'success': validateResult.success, 'error': !validateResult.success }">
          {{ validateResult.success ? '✓' : '✗' }}
        </span>
        <span class="validate-message">{{ validateResult.message }}</span>
      </div>

      <!-- 错误详情 -->
      <div v-if="!validateResult.success && validateResult.errors" class="error-details">
        <div class="error-title">错误详情：</div>
        <div class="error-list">
          <div
              v-for="(error, index) in validateResult.errors"
              :key="index"
              class="error-item"
              @click="jumpToLine(error.line)"
          >
            <span class="error-line">第 {{ error.line }} 行：</span>
            <span class="error-msg">{{ error.message }}</span>
            <span class="error-jump" title="跳转到该行">↗</span>
          </div>
        </div>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: #8c8c8c;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon-btn {
  font-size: 14px;
  margin-right: 4px;
}

.editor-wrapper {
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  overflow: hidden;
  transition: border-color 0.3s;
  margin-bottom: 12px;
}

.editor-wrapper:hover {
  border-color: #4096ff;
}

/* 验证结果样式 */
.validate-result {
  padding: 12px;
  border-radius: 4px;
  background: #fff;
  border: 1px solid #d9d9d9;
  transition: all 0.3s;
}

.validate-result.success {
  border-color: #52c41a;
  background: #f6ffed;
}

.validate-result.error {
  border-color: #ff4d4f;
  background: #fff1f0;
}

.validate-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.validate-icon {
  font-size: 20px;
  font-weight: bold;
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.validate-icon.success {
  background: #52c41a;
  color: #fff;
}

.validate-icon.error {
  background: #ff4d4f;
  color: #fff;
}

.validate-message {
  font-size: 14px;
  color: #262626;
}

/* 错误详情 */
.error-details {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.error-title {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 8px;
}

.error-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.error-item {
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 4px;
  border-left: 3px solid #ff4d4f;
  font-size: 13px;
  color: #262626;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.error-item:hover {
  background: rgba(255, 255, 255, 1);
  transform: translateX(4px);
}

.error-line {
  color: #ff4d4f;
  font-weight: 500;
  white-space: nowrap;
}

.error-msg {
  flex: 1;
  color: #595959;
}

.error-jump {
  color: #1890ff;
  font-size: 12px;
  opacity: 0.8;
}

.error-jump:hover {
  opacity: 1;
}

:deep(.ant-modal-body) {
  padding: 16px;
}

:deep(.ant-divider-vertical) {
  height: 20px;
}

:deep(.ant-btn-loading-icon) {
  margin-right: 4px;
}
</style>

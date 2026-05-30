<script setup lang="ts">
import {reactive, ref, watch} from "vue";
import {message} from "ant-design-vue";

import {ImportDatabaseTables, SetDBConfig, TestDatabaseConnection} from "../../../wailsjs/go/main/App";

const props = defineProps<{
  open?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'success', value: typeof formData): void
}>()

const innerOpen = ref(false)

// 同步外部的 open 状态
watch(() => props.open, (val) => {
  innerOpen.value = val ?? false
}, {immediate: true})

const formRef = ref(); // 表单引用，用于验证
const testLoading = ref(false); // 测试连接按钮的加载状态
const importLoading = ref(false); // 导入按钮的加载状态

const formData = reactive({
  dbType: 'mysql',
  dsn: ''
})

const dbTypes = [
  {value: 'mysql', label: 'MySQL'},
  {value: 'postgresql', label: 'PostgreSQL'},
  {value: 'sqlite', label: 'SQLite'},
  {value: 'oracle', label: 'Oracle'}
]

const formRules = {
  dsn: [
    {required: true, message: '请输入数据源名称(DSN)', trigger: 'blur'},
    {min: 5, message: 'DSN长度至少5个字符', trigger: 'blur'}
  ]
}

// 关闭模态框
function handleClose() {
  emit('update:open', false);
  resetForm();
}

// 提交表单
async function handleCommit() {
  try {
    // 表单验证
    await formRef.value.validate();

    importLoading.value = true;

    const res = await ImportDatabaseTables({
      useDSN: true,
      dsn: formData['dsn'] || '',
      type: formData['dbType'] || '',
      host: "",
      port: 0,
      database: "",
      username: "",
      password: "",
      ssl: false,
      dbPath: ""
    });
    if (res !== '') {
      message.error(`数据库导入失败：${res}`);
      return;
    }

    await SetDBConfig({
      database: "", dbPath: "", host: "", password: "", port: 0, ssl: false, username: "",
      dsn: formData['dsn'] || '',
      type: formData['dbType'] || '',
      useDSN: true
    });

    message.success('数据库导入成功！');

    emit('success', {...formData}); // 触发成功事件，传递数据
    handleClose(); // 验证成功后关闭
  } catch (error) {
    console.error('表单验证失败:', error);
    message.error('请检查表单信息');
  } finally {
    importLoading.value = false;
  }
}

// 测试数据库连接
async function testConnection() {
  try {
    // 验证 DSN 字段
    await formRef.value.validateFields(['dsn']);

    testLoading.value = true;
    console.log('测试数据库连接:', formData);

    const result = await TestDatabaseConnection({
      useDSN: true,
      dsn: formData['dsn'] || '',
      type: formData['dbType'] || '',
      host: "",
      port: 0,
      database: "",
      username: "",
      password: "",
      ssl: false,
      dbPath: ""
    });
    // console.log(result);
    if (result && result.success) {
      message.success('数据库连接成功！');
    } else {
      message.error(result?.message || '数据库连接失败，请检查DSN信息');
    }
    return result.success;
  } catch (error) {
    console.error('连接测试失败:', error);
    message.error('请检查DSN信息');
    return false;
  } finally {
    testLoading.value = false;
  }
}

// 重置表单
function resetForm() {
  formData.dbType = 'mysql';
  formData.dsn = '';

  formRef.value?.resetFields();
}
</script>

<template>
  <a-modal
      v-model:open="innerOpen"
      title="数据库配置"
      :width="600"
      @ok="handleCommit"
      @cancel="handleClose"
      okText="导入"
      cancelText="取消"
  >
    <template #footer>
      <a-button @click="handleClose">取消</a-button>
      <a-button type="primary" :loading="testLoading" @click="testConnection">测试连接</a-button>
      <a-button type="primary" :loading="importLoading" :disabled="!formData.dsn.trim()" @click="handleCommit">导入
      </a-button>
    </template>
    <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        layout="vertical"
        class="db-config-form"
    >
      <!-- 数据库类型选择 -->
      <a-form-item label="数据库类型" name="dbType">
        <a-select v-model:value="formData.dbType" style="width: 100%">
          <a-select-option v-for="dbType in dbTypes" :key="dbType.value" :value="dbType.value">
            {{ dbType.label }}
          </a-select-option>
        </a-select>
      </a-form-item>

      <!-- DSN连接字符串 -->
      <a-form-item label="数据源名称(DSN)" name="dsn">
        <a-textarea
            v-model:value="formData.dsn"
            placeholder="示例: mysql://user:password@localhost:3306/dbname?charset=utf8mb4"
            :rows="3"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped>
.db-config-form {
  padding: 16px 0;
}

:deep(.ant-form-item) {
  margin-bottom: 16px;
}
</style>

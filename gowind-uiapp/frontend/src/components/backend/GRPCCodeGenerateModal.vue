<script setup lang="ts">
import {ref, watch, reactive} from "vue";
import {message} from "ant-design-vue";
import {GenerateGrpcCode} from "../../../wailsjs/go/main/App";

const props = defineProps<{
  open?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'success', value: { ormType: string }): void
}>()

const innerOpen = ref(false)
const formRef = ref()
const confirmLoading = ref(false)

const formData = reactive({
  ormType: 'ent'
})

const ormTypes = [
  {value: 'ent', label: 'Ent'},
  {value: 'gorm', label: 'GORM'}
]

const formRules = {
  ormType: [
    {required: true, message: '请选择 ORM 类型', trigger: 'change'}
  ]
}

// 同步外部的 open 状态
watch(() => props.open, (val) => {
  innerOpen.value = val ?? false
}, {immediate: true})

// 关闭模态框
function handleClose() {
  emit('update:open', false)
  resetForm()
}

// 重置表单
function resetForm() {
  formData.ormType = 'ent'
  formRef.value?.clearValidate()
}

// 提交表单
async function handleCommit() {
  try {
    // 表单验证
    await formRef.value.validate()

    confirmLoading.value = true

    const res = await GenerateGrpcCode(formData.ormType);
    if (res == "") {
      message.success('代码生成成功');

      emit('success', {ormType: formData.ormType})
    } else {
      message.error('代码生成失败: ' + res)
    }

    handleClose()
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    confirmLoading.value = false
  }
}
</script>

<template>
  <a-modal
      v-model:open="innerOpen"
      title="生成gRPC服务"
      :width="500"
      @cancel="handleClose"
  >
    <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 16 }"
    >
      <a-form-item label="ORM 类型" name="ormType">
        <a-select
            v-model:value="formData.ormType"
            placeholder="请选择 ORM 类型"
        >
          <a-select-option
              v-for="item in ormTypes"
              :key="item.value"
              :value="item.value"
          >
            {{ item.label }}
          </a-select-option>
        </a-select>
      </a-form-item>
    </a-form>

    <template #footer>
      <a-button @click="handleClose">取消</a-button>
      <a-button
          type="primary"
          :loading="confirmLoading"
          @click="handleCommit"
      >
        确定
      </a-button>
    </template>
  </a-modal>
</template>

<style scoped>

</style>

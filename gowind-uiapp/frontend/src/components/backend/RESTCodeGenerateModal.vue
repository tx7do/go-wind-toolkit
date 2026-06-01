<script setup lang="ts">
import {ref, watch, reactive} from "vue";
import {message} from "ant-design-vue";
import {GenerateRestCode} from "../../../wailsjs/go/main/App";

const props = defineProps<{
  open?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'success', value: { serviceName: string }): void
}>()

const innerOpen = ref(false)
const formRef = ref()
const confirmLoading = ref(false)

const formData = reactive({
  serviceName: 'admin'
})

const formRules = {
  serviceName: [
    {required: true, message: '请输入REST服务名', trigger: 'change'}
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
  formData.serviceName = 'admin'
  formRef.value?.clearValidate()
}

// 提交表单
async function handleCommit() {
  try {
    // 表单验证
    await formRef.value.validate()

    confirmLoading.value = true

    const res = await GenerateRestCode(formData.serviceName, "per-table");
    if (res == "") {
      message.success('代码生成成功');

      emit('success', {serviceName: formData.serviceName})
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
      title="生成BFF服务"
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
      <a-form-item label="REST服务名" name="serviceName">
        <a-input
            v-model:value="formData.serviceName">
        </a-input>
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

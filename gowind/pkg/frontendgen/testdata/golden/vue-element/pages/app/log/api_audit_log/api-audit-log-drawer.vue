<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    :size="DRAWER_WIDTH"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.apiAuditLog.duration')" prop="duration">
        <ElInput
          v-model="formData.duration"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.apiAuditLog.ipAddress')" prop="ipAddress">
        <ElInput
          v-model="formData.ipAddress"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.apiAuditLog.isSuccess')" prop="isSuccess">
        <ElSwitch v-model="formData.isSuccess" />
      </ElFormItem>

      <ElFormItem :label="$t('pages.apiAuditLog.operatorName')" prop="operatorName">
        <ElInput
          v-model="formData.operatorName"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.apiAuditLog.requestMethod')" prop="requestMethod">
        <ElSelect v-model="formData.requestMethod" placeholder="请求方法">
          <ElOption v-for="item in requestMethodList" :key="item.value" :label="item.label" :value="item.value" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.apiAuditLog.requestPath')" prop="requestPath">
        <ElInput
          v-model="formData.requestPath"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();



const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 请求方法 选项(取自 OpenAPI enum)
const requestMethodList = [
  { label: "GET", value: "GET" },
  { label: "POST", value: "POST" },
  { label: "PUT", value: "PUT" },
  { label: "DELETE", value: "DELETE" },
];

// 表单数据
const formData = reactive({
  duration: 1,
  ipAddress: "",
  isSuccess: false,
  operatorName: "",
  requestMethod: "",
  requestPath: "",
});

// 表单验证规则
const formRules = {
  duration: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  ipAddress: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  operatorName: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  requestMethod: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  requestPath: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.apiAuditLog.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.apiAuditLog.moduleName") })
);

// 打开抽屉
function open(row?) {
  visible.value = true;

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentId.value = row.id;
    Object.assign(formData, row);
  } else {
    // 创建模式
    isCreate.value = true;
    currentId.value = undefined;
    resetForm();
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.duration = 1;
  formData.ipAddress = "";
  formData.isSuccess = false;
  formData.operatorName = "";
  formData.requestMethod = "";
  formData.requestPath = "";

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };


    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 不是验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法给父组件
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>

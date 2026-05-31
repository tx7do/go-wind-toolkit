// Vue3 Element Plus - 页面代码生成器
// 生成 pages/index.vue 和 drawer.vue
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { toCamelCase, toPascalCase, toKebabCase, getCrudPaths } from '../../utils/openapi-parser'

export interface PageTemplateOptions {
  service: ParsedService
  /** 模块路径名（如 permission/role） */
  modulePath?: string
  /** i18n 模块名（如 pages.role） */
  i18nModuleKey?: string
  /** 模块显示名称 */
  moduleDisplayName?: string
}

/**
 * 根据字段类型推断表单组件类型
 */
function getFieldFormType(field: ParsedField): {
  type: string
  component: string
  attrs: string
} {
  if (field.isBoolean) {
    return { type: 'switch', component: 'ElSwitch', attrs: '' }
  }
  if (field.isEnum && field.enumValues) {
    if (field.enumValues.length <= 3) {
      return {
        type: 'radio',
        component: 'ElRadioGroup',
        attrs: '',
      }
    }
    return {
      type: 'select',
      component: 'ElSelect',
      attrs: `\n          <ElOption v-for="item in ${field.name}List" :key="item.value" :label="item.label" :value="item.value" />`,
    }
  }
  if (field.isInteger && field.name.toLowerCase().includes('sort')) {
    return { type: 'input-number', component: 'ElInputNumber', attrs: '          :min="1"\n          style="width: 100%"' }
  }
  if (field.isDate) {
    return { type: 'date-picker', component: 'ElDatePicker', attrs: '          type="datetime"\n          style="width: 100%"' }
  }
  if (field.isArray) {
    return { type: 'input', component: 'ElInput', attrs: '' }
  }
  if (field.name.toLowerCase().includes('description') || field.name.toLowerCase().includes('remark')) {
    return { type: 'textarea', component: 'ElInput', attrs: '          type="textarea"\n          :rows="3"' }
  }
  return { type: 'input', component: 'ElInput', attrs: '' }
}

/**
 * 根据字段类型推断表格列显示方式
 */
function getColumnDisplay(field: ParsedField): {
  cellType?: string
  slotName?: string
  dateFormat?: string
} {
  if (field.isBoolean) {
    return { slotName: field.name }
  }
  if (field.isEnum) {
    return { slotName: field.name }
  }
  if (field.isDate) {
    return { cellType: 'date', dateFormat: 'YYYY-MM-DD HH:mm:ss' }
  }
  return {}
}

/**
 * 获取搜索字段（排除 id、数组、长文本等字段）
 */
function getSearchFields(fields: ParsedField[]): ParsedField[] {
  const skipFields = new Set(['id', 'description', 'remark', 'sortOrder', 'isDefault', 'isEnabled'])
  return fields.filter(f => !skipFields.has(f.name) && !f.isArray && !f.isDate)
}

/**
 * 生成列表页面 index.vue
 */
export function generatePageCode(options: PageTemplateOptions): string {
  const {
    service,
    modulePath = service.kebabName,
    i18nModuleKey = `pages.${toCamelCase(service.modelName)}`,
    moduleDisplayName = service.description,
  } = options

  const modelCamel = toCamelCase(service.modelName)
  const modelPascal = toPascalCase(service.modelName)
  const crudPaths = getCrudPaths(service)
  const hasList = !!crudPaths.list
  const hasDelete = !!crudPaths.delete

  if (!hasList) return `// ${service.tagName} 没有 List 操作，无法生成列表页面`

  const searchFields = getSearchFields(service.fields)
  const tableFields = service.fields.filter(f =>
    !['id'].includes(f.name) && !f.isArray
  )

  // 搜索字段配置
  const searchFieldsCode = searchFields.map(f => {
    return `      {
        type: "input",
        label: $t("${i18nModuleKey}.${f.name}"),
        field: "${f.name}",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },`
  }).join('\n')

  // 表格列配置
  const columnCodes: string[] = []
  columnCodes.push(`      { type: "index", label: $t("common.table.seq"), width: 60 },`)

  for (const field of tableFields) {
    const display = getColumnDisplay(field)
    const col: string[] = []
    col.push(`      {`)

    // 第一个字段固定在左侧
    if (columnCodes.length === 1) {
      col.push(`        prop: "${field.name}",`)
      col.push(`        label: $t("${i18nModuleKey}.${field.name}"),`)
      col.push(`        minWidth: 120,`)
      col.push(`        fixed: "left",`)
    } else if (field.isBoolean || field.isEnum) {
      col.push(`        prop: "${field.name}",`)
      col.push(`        label: $t("${i18nModuleKey}.${field.name}"),`)
      col.push(`        width: 100,`)
      if (display.slotName) {
        col.push(`        slotName: "${display.slotName}",`)
      }
    } else if (field.isDate) {
      col.push(`        prop: "${field.name}",`)
      col.push(`        label: $t("${i18nModuleKey}.${field.name}"),`)
      col.push(`        minWidth: 160,`)
      if (display.cellType) col.push(`        cellType: "${display.cellType}",`)
      if (display.dateFormat) col.push(`        dateFormat: "${display.dateFormat}",`)
    } else if (field.isInteger && field.name.toLowerCase().includes('sort')) {
      col.push(`        prop: "${field.name}",`)
      col.push(`        label: $t("common.table.sortOrder"),`)
      col.push(`        width: 100,`)
      col.push(`        align: "right",`)
    } else {
      col.push(`        prop: "${field.name}",`)
      col.push(`        label: $t("${i18nModuleKey}.${field.name}"),`)
      col.push(`        minWidth: 120,`)
    }

    col.push(`      },`)
    columnCodes.push(col.join('\n'))
  }

  // 操作列
  columnCodes.push(`      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "delete", label: $t("common.button.delete"), icon: "lucide:trash-2", attrs: { type: "danger" } },
        ],
      },`)

  // 模板中的 slot 定义
  const slotCodes: string[] = []
  for (const field of tableFields) {
    if (field.isBoolean) {
      slotCodes.push(`      <!-- ${field.description} -->
      <template #${field.name}="scope">
        <ElTag size="small" :type="scope.row.${field.name} ? 'success' : 'info'" effect="plain">
          {{ enableBoolToName(scope.row.${field.name}) }}
        </ElTag>
      </template>`)
    } else if (field.isEnum && field.enumValues) {
      slotCodes.push(`      <!-- ${field.description} -->
      <template #${field.name}="scope">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.${field.name})">
          {{ statusToName(scope.row.${field.name}) }}
        </ElTag>
      </template>`)
    }
  }

  // composable imports
  const composableImports: string[] = ['enableBoolToName']
  if (hasList) composableImports.push(`fetchList${modelPascal}s`)
  if (hasDelete) composableImports.push(`useDelete${modelPascal}`)
  // check if we need status helpers
  const hasStatusEnum = service.fields.some(f => f.isEnum && f.name === 'status')
  if (hasStatusEnum) {
    composableImports.push('statusToColor', 'statusToName', 'statusList')
  }

  let code = `<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
${slotCodes.join('\n\n')}
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    ${/* 使用 PascalCase 组件名 */''}
    <${modelPascal}Drawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ${modelPascal}Drawer from "./${service.kebabName}-drawer.vue";

import {
  ${composableImports.join(',\n  ')},
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
${hasDelete ? `const { mutateAsync: delete${modelPascal} } = useDelete${modelPascal}();\n` : ''}
const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
${searchFieldsCode}
    ],
  },

  table: {
    listAction: async (query) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchList${modelPascal}s(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },${hasDelete ? `
    deleteAction: async (ids: string) => {
      await delete${modelPascal}({ id: ids as any });
    },` : ''}
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
${columnCodes.join('\n')}
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleEdit(row) {
  drawerRef.value?.open(row);
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
`

  return code
}

/**
 * 生成抽屉表单 drawer.vue
 */
export function generateDrawerCode(options: PageTemplateOptions): string {
  const {
    service,
    i18nModuleKey = `pages.${toCamelCase(service.modelName)}`,
    moduleDisplayName = service.description,
  } = options

  const modelCamel = toCamelCase(service.modelName)
  const modelPascal = toPascalCase(service.modelName)
  const crudPaths = getCrudPaths(service)
  const hasCreate = !!crudPaths.create
  const hasUpdate = !!crudPaths.update

  // 表单字段（排除系统字段和ID）
  const formFields = service.fields.filter(f =>
    !['id'].includes(f.name) && !f.isDate
  )

  // 表单项模板代码
  const formItemCodes = formFields.map(f => {
    const formType = getFieldFormType(f)
    const label = `$t('${i18nModuleKey}.${f.name}')`
    const prop = `prop="${f.name}"`

    if (formType.type === 'switch') {
      return `      <ElFormItem :label="${label}" ${prop}>
        <${formType.component} v-model="formData.${f.name}" />
      </ElFormItem>`
    }

    if (formType.type === 'radio') {
      const radioButtons = (f.enumValues || []).map(v =>
        `          <ElRadioButton :value="'${v}'">{{ $t("enum.${f.name}.${v}") }}</ElRadioButton>`
      ).join('\n')
      return `      <ElFormItem :label="${label}" ${prop}>
        <ElRadioGroup v-model="formData.${f.name}">
${radioButtons}
        </ElRadioGroup>
      </ElFormItem>`
    }

    if (formType.type === 'select') {
      return `      <ElFormItem :label="${label}" ${prop}>
        <ElSelect v-model="formData.${f.name}" placeholder="${f.description}">
          <ElOption v-for="item in ${f.name}List" :key="item.value" :label="item.label" :value="item.value" />
        </ElSelect>
      </ElFormItem>`
    }

    if (formType.type === 'input-number') {
      return `      <ElFormItem :label="${label}" ${prop}>
        <ElInputNumber
          v-model="formData.${f.name}"
${formType.attrs}
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>`
    }

    // 默认 input / textarea
    return `      <ElFormItem :label="${label}" ${prop}>
        <${formType.component}
          v-model="formData.${f.name}"
${formType.attrs ? formType.attrs + '\n' : ''}          :placeholder="$t('common.placeholder.input')"
          ${formType.type !== 'textarea' ? 'clearable' : ''}
        />
      </ElFormItem>`
  }).join('\n\n')

  // formData 字段默认值
  const formDataDefaults = formFields.map(f => {
    if (f.isBoolean) return `  ${f.name}: ${f.name === 'isEnabled' || f.name === 'isDefault' ? 'true' : 'false'},`
    if (f.isInteger) return `  ${f.name}: 1,`
    if (f.isArray) return `  ${f.name}: [] as any[],`
    return `  ${f.name}: "",`
  }).join('\n')

  // formRules
  const requiredFields = formFields.filter(f =>
    !f.isBoolean && !['sortOrder', 'description', 'remark'].includes(f.name)
  )
  const formRulesCode = requiredFields.map(f =>
    `  ${f.name}: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],`
  ).join('\n')

  // resetForm
  const resetFormCode = formFields.map(f => {
    if (f.isBoolean) return `  formData.${f.name} = ${f.name === 'isEnabled' || f.name === 'isDefault' ? 'true' : 'false'};`
    if (f.isInteger) return `  formData.${f.name} = 1;`
    if (f.isArray) return `  formData.${f.name} = [];`
    return `  formData.${f.name} = "";`
  }).join('\n')

  // composable imports
  const composableImports: string[] = []
  if (hasCreate) composableImports.push(`useCreate${modelPascal}`)
  if (hasUpdate) composableImports.push(`useUpdate${modelPascal}`)

  let code = `<template>
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

${formItemCodes}
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

import {
  ${composableImports.join(',\n  ')},
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();
${hasCreate ? `const { mutateAsync: create${modelPascal} } = useCreate${modelPascal}();` : ''}
${hasUpdate ? `const { mutateAsync: update${modelPascal} } = useUpdate${modelPascal}();` : ''}

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
${formDataDefaults}
});

// 表单验证规则
const formRules = {
${formRulesCode}
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("${i18nModuleKey}.moduleName") })
    : $t("common.modal.update", { moduleName: $t("${i18nModuleKey}.moduleName") })
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
${resetFormCode}

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };
${hasCreate ? `
    if (isCreate.value) {
      await create${modelPascal}(values);
      ElMessage.success($t("common.notification.createSuccess"));
    }` : ''}${hasUpdate ? `${hasCreate ? ' else' : ''} {
      await update${modelPascal}({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }` : ''}

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
`

  return code
}

/**
 * Vue Vben - 页面代码生成器
 * 生成 VxeGrid 列表页 index.vue 和 useVbenDrawer/useVbenForm 编辑抽屉 xxx-drawer.vue
 */
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { toPascalCase, toCamelCase, getCrudPaths } from '../../utils/openapi-parser'

export interface VbenPageTemplateOptions {
  service: ParsedService
  modulePath?: string
  serviceName?: string
}

// ==============================
// 字段类型推断辅助函数
// ==============================

/**
 * 获取搜索表单的字段（排除 id、日期、布尔等不适合搜索的字段）
 */
function getSearchFormFields(fields: ParsedField[]): ParsedField[] {
  const skipFields = new Set(['id', 'description', 'remark', 'sortOrder', 'isDefault', 'isEnabled', 'createdAt', 'updatedAt'])
  return fields.filter(f => !skipFields.has(f.name) && !f.isArray && !f.isDate && !f.isBoolean)
}

/**
 * 获取表格展示字段（排除 id、数组类型）
 */
function getTableFields(fields: ParsedField[]): ParsedField[] {
  return fields.filter(f => !['id'].includes(f.name) && !f.isArray)
}

/**
 * 获取表单编辑字段（排除 id、日期字段）
 */
function getFormFields(fields: ParsedField[]): ParsedField[] {
  return fields.filter(f => !['id', 'createdAt', 'updatedAt'].includes(f.name) && !f.isDate)
}

/**
 * 根据字段推断 VxeGrid 列定义中的 slots
 */
function getColumnSlots(field: ParsedField): string | null {
  if (field.isBoolean) return field.name
  if (field.isEnum && field.name.toLowerCase().includes('status')) return 'status'
  return null
}

/**
 * 根据字段推断 useVbenForm schema 的 component 类型
 */
function getFormComponent(field: ParsedField): { component: string; extraProps: string; defaultValue?: string } {
  if (field.isBoolean) {
    const defaultVal = ['isEnabled', 'isDefault'].includes(field.name) ? 'true' : 'false'
    return {
      component: 'Switch',
      extraProps: `        class: 'w-auto',`,
      defaultValue: defaultVal,
    }
  }
  if (field.isEnum && field.name.toLowerCase().includes('status')) {
    return {
      component: 'RadioGroup',
      extraProps: `        optionType: 'button',\n        buttonStyle: 'solid',\n        class: 'flex flex-wrap',\n        options: statusList,`,
      defaultValue: "'ON'",
    }
  }
  if (field.isInteger && field.name.toLowerCase().includes('sort')) {
    return {
      component: 'InputNumber',
      extraProps: `        placeholder: $t('ui.placeholder.input'),\n        allowClear: true,`,
      defaultValue: '1',
    }
  }
  if (field.name.toLowerCase().includes('description') || field.name.toLowerCase().includes('remark')) {
    return {
      component: 'Textarea',
      extraProps: `        placeholder: $t('ui.placeholder.input'),\n        allowClear: true,`,
    }
  }
  return {
    component: 'Input',
    extraProps: `        placeholder: $t('ui.placeholder.input'),\n        allowClear: true,`,
  }
}

/**
 * 生成 formSchema 中的 rules
 */
function getFormRules(field: ParsedField): string | null {
  if (field.isBoolean) return null
  if (['sortOrder', 'description', 'remark'].includes(field.name)) return null
  if (field.isEnum && field.name.toLowerCase().includes('status')) return "'selectRequired'"
  return "'required'"
}

// ==============================
// 列表页面 index.vue
// ==============================

export function generateVbenPageCode(options: VbenPageTemplateOptions): string {
  const {
    service,
    modulePath = service.kebabName,
    serviceName = 'admin',
  } = options

  const modelPascal = toPascalCase(service.modelName)
  const modelCamel = toCamelCase(service.modelName)
  const fileName = service.kebabName.replace(/-/g, '')
  const crudPaths = getCrudPaths(service)
  const hasList = !!crudPaths.list
  const hasDelete = !!crudPaths.delete

  if (!hasList) return `<!-- ${service.tagName} 没有 List 操作，无法生成列表页面 -->`

  const hasStatusEnum = service.fields.some(f => f.isEnum && f.name.toLowerCase().includes('status'))
  const hasBoolField = service.fields.some(f => f.isBoolean)

  // 搜索表单字段
  const searchFields = getSearchFormFields(service.fields)
  const formSchemaItems = searchFields.map(f => {
    const isStatusEnum = f.isEnum && f.name.toLowerCase().includes('status')
    const component = isStatusEnum ? 'Select' : 'Input'
    let extraProps = `        placeholder: $t('ui.placeholder.${isStatusEnum ? 'select' : 'input'}'),\n        allowClear: true,`
    if (isStatusEnum) {
      extraProps += `\n        options: statusList,\n        showSearch: true,\n        filterOption: (input: string, option: any) =>\n          option.label.toLowerCase().includes(input.toLowerCase()),`
    }
    return `    {
      component: '${component}',
      fieldName: '${f.name}',
      label: $t('page.${modelCamel}.${f.name}'),
      componentProps: {
${extraProps}
      },
    },`
  }).join('\n')

  // 表格列
  const tableFields = getTableFields(service.fields)
  const columnItems: string[] = []
  // 序号列
  columnItems.push(`    { title: $t('ui.table.seq'), type: 'seq', width: 50 },`)

  for (const field of tableFields) {
    const lower = field.name.toLowerCase()
    const slots = getColumnSlots(field)
    const isDate = field.isDate

    if (field.isBoolean) {
      columnItems.push(`    {
      title: $t('page.${modelCamel}.${field.name}'),
      field: '${field.name}',
      slots: { default: '${field.name}' },
      minWidth: 50,
    },`)
    } else if (isDate) {
      columnItems.push(`    {
      title: $t('ui.table.${field.name === 'createdAt' ? 'createdAt' : field.name === 'updatedAt' ? 'updatedAt' : field.name}'),
      field: '${field.name}',
      formatter: 'formatDateTime',
      minWidth: 140,
    },`)
    } else if (field.isEnum && lower.includes('status')) {
      columnItems.push(`    {
      title: $t('ui.table.status'),
      field: '${field.name}',
      slots: { default: 'status' },
      width: 95,
    },`)
    } else if (lower.includes('description') || lower.includes('remark')) {
      columnItems.push(`    {
      title: $t('ui.table.description'),
      field: '${field.name}',
      minWidth: 120,
    },`)
    } else if (field.isInteger && lower.includes('sort')) {
      columnItems.push(`    {
      title: $t('ui.table.sortOrder'),
      field: '${field.name}',
      minWidth: 100,
    },`)
    } else {
      columnItems.push(`    {
      title: $t('page.${modelCamel}.${field.name}'),
      field: '${field.name}',
      minWidth: 120,
    },`)
    }
  }

  // 操作列
  columnItems.push(`    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 90,
    },`)

  // 模板中的布尔字段 slot
  const boolSlotTemplates = service.fields
    .filter(f => f.isBoolean)
    .map(f => `      <template #${f.name}="{ row }">
        <a-tag :color="enableBoolToColor(row.${f.name})">
          {{ enableBoolToName(row.${f.name}) }}
        </a-tag>
      </template>`)
    .join('\n')

  // status slot 模板
  const statusSlotTemplate = hasStatusEnum
    ? `      <template #status="{ row }">
        <a-tag :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </a-tag>
      </template>`
    : ''

  // 导入列表
  const typeImport = `type ${fileName}_${modelPascal} as ${modelPascal}`
  const apiImports = ['fetchList' + modelPascal + 's', 'PaginationQuery']
  if (hasDelete) apiImports.push('useDelete' + modelPascal)
  if (hasStatusEnum) {
    apiImports.push('statusList', 'statusToColor', 'statusToName')
  }
  if (hasBoolField) {
    apiImports.push('enableBoolToColor', 'enableBoolToName')
  }

  let code = `<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { ${typeImport} } from '#/api';
import {
  ${apiImports.join(',\n  ')},
} from '#/api';
import { $t } from '#/locales';

import ${modelPascal}Drawer from './${fileName}-drawer.vue';
`
  if (hasDelete) {
    code += `
const { mutateAsync: delete${modelPascal} } = useDelete${modelPascal}();
`
  }

  code += `
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
${formSchemaItems}
  ],
};

const gridOptions: VxeGridProps<${modelPascal}> = {
  height: 'auto',
  stripe: false,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchList${modelPascal}s(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
${columnItems.join('\n')}
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: ${modelPascal}Drawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });
  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}
`
  if (hasDelete) {
    code += `
/* 删除 */
async function handleDelete(row: any) {
  try {
    await delete${modelPascal}({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}
`
  }

  code += `</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.${modulePath.replace(/\//g, '.')}.${modelCamel}')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.${modelCamel}.button.create') }}
        </a-button>
      </template>
${boolSlotTemplates ? boolSlotTemplates + '\n' : ''}${statusSlotTemplate ? statusSlotTemplate + '\n' : ''}      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.${modelCamel}.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
`
  return code
}

// ==============================
// 编辑抽屉 xxx-drawer.vue
// ==============================

export function generateVbenDrawerCode(options: VbenPageTemplateOptions): string {
  const {
    service,
    serviceName = 'admin',
  } = options

  const modelPascal = toPascalCase(service.modelName)
  const modelCamel = toCamelCase(service.modelName)
  const fileName = service.kebabName.replace(/-/g, '')
  const crudPaths = getCrudPaths(service)
  const hasCreate = !!crudPaths.create
  const hasUpdate = !!crudPaths.update

  const formFields = getFormFields(service.fields)
  const hasStatusEnum = formFields.some(f => f.isEnum && f.name.toLowerCase().includes('status'))

  // 生成 form schema
  const schemaItems = formFields.map(f => {
    const form = getFormComponent(f)
    const rules = getFormRules(f)

    let item = `    {
      component: '${form.component}',
      fieldName: '${f.name}',
      label: $t('page.${modelCamel}.${f.name}'),`

    if (form.defaultValue !== undefined) {
      item += `\n      defaultValue: ${form.defaultValue},`
    }
    if (rules) {
      item += `\n      rules: ${rules},`
    }
    item += `\n      componentProps: {
${form.extraProps}
      },
    },`
    return item
  }).join('\n')

  const apiImports: string[] = []
  if (hasCreate) apiImports.push('useCreate' + modelPascal)
  if (hasUpdate) apiImports.push('useUpdate' + modelPascal)

  let code = `<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  ${apiImports.join(',\n  ')},
} from '#/api';
`
  if (hasStatusEnum) {
    code += `import { statusList } from '#/api';\n`
  }

  if (hasCreate) {
    code += `const { mutateAsync: create${modelPascal} } = useCreate${modelPascal}();\n`
  }
  if (hasUpdate) {
    code += `const { mutateAsync: update${modelPascal} } = useUpdate${modelPascal}();\n`
  }

  code += `
const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('page.${modelCamel}.button.create')
    : $t('page.${modelCamel}.button.update'),
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
${schemaItems}
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    const values = await baseFormApi.getValues();

    try {
      await (data.value?.create
        ? create${modelPascal}(values)
        : update${modelPascal}({ id: data.value.row.id, values }));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });
    } catch {
      notification.error({
        message: data.value?.create
          ? $t('ui.notification.create_failed')
          : $t('ui.notification.update_failed'),
      });
    } finally {
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();

      if (data.value.row !== undefined) {
        baseFormApi.setValues(data.value?.row);
      }

      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
`
  return code
}

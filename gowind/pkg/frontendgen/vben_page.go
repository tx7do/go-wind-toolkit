package frontendgen

import "strings"

// ==============================
// 字段筛选/推断辅助（vben）
// ==============================

var vbenSearchSkipFields = map[string]bool{
	"id": true, "description": true, "remark": true, "sortOrder": true,
	"isDefault": true, "isEnabled": true, "createdAt": true, "updatedAt": true,
}

// vbenSearchFormFields 搜索表单字段（排除不适合搜索的字段）
func vbenSearchFormFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if !vbenSearchSkipFields[f.Name] && !f.IsArray && !f.IsDate && !f.IsBoolean {
			out = append(out, *f)
		}
	}
	return out
}

// vbenTableFields 表格展示字段（排除 id、数组类型）
func vbenTableFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if f.Name != "id" && !f.IsArray {
			out = append(out, *f)
		}
	}
	return out
}

// vbenFormFields 表单编辑字段（排除 id、系统时间字段与日期字段）
func vbenFormFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if f.Name != "id" && f.Name != "createdAt" && f.Name != "updatedAt" && !f.IsDate {
			out = append(out, *f)
		}
	}
	return out
}

func isStatusField(field *ParsedField) bool {
	return field.IsEnum && strings.Contains(strings.ToLower(field.Name), "status")
}

func anyStatusField(fields []ParsedField) bool {
	for i := range fields {
		if isStatusField(&fields[i]) {
			return true
		}
	}
	return false
}

func anyBoolField(fields []ParsedField) bool {
	for i := range fields {
		if fields[i].IsBoolean {
			return true
		}
	}
	return false
}

// ==============================
// 列表页 index.vue
// ==============================

// vbenPageCode 生成 VxeGrid 列表页（对应 TS 版 page-template.ts generateVbenPageCode）
func vbenPageCode(service *ParsedService, serviceName, modulePath string) string {
	modelPascal := toPascalCase(service.ModelName)
	modelCamel := toCamelCase(service.ModelName)
	fileName := service.KebabName
	prefix := service.TypePrefix

	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	hasDelete := crudPaths.Delete != nil
	if !hasList {
		return "<!-- " + service.TagName + " 没有 List 操作，无法生成列表页面 -->"
	}

	hasStatusEnum := anyStatusField(service.Fields)
	hasBool := anyBoolField(service.Fields)

	// 搜索表单 schema
	var formSchemaItems []string
	searchFields := vbenSearchFormFields(service.Fields)
	for i := range searchFields {
		f := &searchFields[i]
		statusEnum := isStatusField(f)
		component := "Input"
		extraProps := "        placeholder: $t('ui.placeholder.input'),\n        allowClear: true,"
		if statusEnum {
			component = "Select"
			extraProps = "        placeholder: $t('ui.placeholder.select'),\n" +
				"        allowClear: true,\n" +
				"        options: statusList,\n" +
				"        showSearch: true,\n" +
				"        filterOption: (input: string, option: any) =>\n" +
				"          option.label.toLowerCase().includes(input.toLowerCase()),"
		}
		formSchemaItems = append(formSchemaItems, "    {\n"+
			"      component: '"+component+"',\n"+
			"      fieldName: '"+f.Name+"',\n"+
			"      label: $t('page."+modelCamel+"."+f.Name+"'),\n"+
			"      componentProps: {\n"+
			extraProps+"\n"+
			"      },\n"+
			"    },")
	}

	// 表格列
	var columnItems []string
	columnItems = append(columnItems, "    { title: $t('ui.table.seq'), type: 'seq', width: 50 },")

	tableFields := vbenTableFields(service.Fields)
	for i := range tableFields {
		field := &tableFields[i]
		lower := strings.ToLower(field.Name)

		switch {
		case field.IsBoolean:
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('page."+modelCamel+"."+field.Name+"'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      slots: { default: '"+field.Name+"' },\n"+
				"      minWidth: 50,\n"+
				"    },")
		case field.IsDate:
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('ui.table."+field.Name+"'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      formatter: 'formatDateTime',\n"+
				"      minWidth: 140,\n"+
				"    },")
		case field.IsEnum && strings.Contains(lower, "status"):
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('ui.table.status'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      slots: { default: 'status' },\n"+
				"      width: 95,\n"+
				"    },")
		case strings.Contains(lower, "description") || strings.Contains(lower, "remark"):
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('ui.table.description'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      minWidth: 120,\n"+
				"    },")
		case field.IsInteger && strings.Contains(lower, "sort"):
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('ui.table.sortOrder'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      minWidth: 100,\n"+
				"    },")
		default:
			columnItems = append(columnItems, "    {\n"+
				"      title: $t('page."+modelCamel+"."+field.Name+"'),\n"+
				"      field: '"+field.Name+"',\n"+
				"      minWidth: 120,\n"+
				"    },")
		}
	}

	columnItems = append(columnItems, "    {\n"+
		"      title: $t('ui.table.action'),\n"+
		"      field: 'action',\n"+
		"      fixed: 'right',\n"+
		"      slots: { default: 'action' },\n"+
		"      width: 90,\n"+
		"    },")

	// 布尔字段 slot 模板
	var boolSlotTemplates []string
	for i := range service.Fields {
		f := &service.Fields[i]
		if f.IsBoolean {
			boolSlotTemplates = append(boolSlotTemplates,
				"      <template #"+f.Name+"=\"{ row }\">\n"+
					"        <a-tag :color=\"enableBoolToColor(row."+f.Name+")\">\n"+
					"          {{ enableBoolToName(row."+f.Name+") }}\n"+
					"        </a-tag>\n"+
					"      </template>")
		}
	}

	statusSlotTemplate := ""
	if hasStatusEnum {
		statusSlotTemplate = "      <template #status=\"{ row }\">\n" +
			"        <a-tag :color=\"statusToColor(row.status)\">\n" +
			"          {{ statusToName(row.status) }}\n" +
			"        </a-tag>\n" +
			"      </template>"
	}

	// 导入列表
	typeImport := "type " + prefix + "_" + modelPascal + " as " + modelPascal
	apiImports := []string{"fetchList" + modelPascal + "s", "PaginationQuery"}
	if hasDelete {
		apiImports = append(apiImports, "useDelete"+modelPascal)
	}
	if hasStatusEnum {
		apiImports = append(apiImports, "statusList", "statusToColor", "statusToName")
	}
	if hasBool {
		apiImports = append(apiImports, "enableBoolToColor", "enableBoolToName")
	}

	var sb strings.Builder
	sb.WriteString(`<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
`)

	_ = serviceName // serviceName 由类型导入路径使用
	sb.WriteString("import { " + typeImport + " } from '#/api';\n")
	sb.WriteString("import {\n  " + strings.Join(apiImports, ",\n  ") + ",\n} from '#/api';\n")
	sb.WriteString("import { $t } from '#/locales';\n\n")
	sb.WriteString("import " + modelPascal + "Drawer from './" + fileName + "-drawer.vue';\n")

	if hasDelete {
		sb.WriteString("\nconst { mutateAsync: delete" + modelPascal + " } = useDelete" + modelPascal + "();\n")
	}

	sb.WriteString(`
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
` + strings.Join(formSchemaItems, "\n") + `
  ],
};

const gridOptions: VxeGridProps<` + modelPascal + `> = {
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
        return await fetchList` + modelPascal + `s(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
` + strings.Join(columnItems, "\n") + `
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: ` + modelPascal + `Drawer,

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
`)

	if hasDelete {
		sb.WriteString(`
/* 删除 */
async function handleDelete(row: any) {
  try {
    await delete` + modelPascal + `({ id: row.id });

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
`)
	}

	sb.WriteString("</script>\n\n<template>\n  <Page auto-content-height>\n")
	sb.WriteString("    <Grid :table-title=\"$t('menu." + strings.ReplaceAll(modulePath, "/", ".") + "." + modelCamel + "')\">\n")
	sb.WriteString(`      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
` + "          {{ $t('page." + modelCamel + ".button.create') }}\n" + `        </a-button>
      </template>
`)
	if len(boolSlotTemplates) > 0 {
		sb.WriteString(strings.Join(boolSlotTemplates, "\n") + "\n")
	}
	if statusSlotTemplate != "" {
		sb.WriteString(statusSlotTemplate + "\n")
	}
	sb.WriteString(`      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
` + "            $t('ui.text.do_you_want_delete', {\n" +
		"              moduleName: $t('page." + modelCamel + ".moduleName'),\n" +
		"            })\n" + `          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
`)

	return sb.String()
}

// ==============================
// 编辑抽屉 xxx-drawer.vue
// ==============================

// vbenDrawerCode 生成 useVbenDrawer/useVbenForm 编辑抽屉
func vbenDrawerCode(service *ParsedService, serviceName string) string {
	modelPascal := toPascalCase(service.ModelName)
	modelCamel := toCamelCase(service.ModelName)

	crudPaths := GetCrudPaths(service)
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil

	formFields := vbenFormFields(service.Fields)
	hasStatusEnum := anyStatusField(formFields)

	// form schema
	var schemaItems []string
	for i := range formFields {
		f := &formFields[i]
		component, extraProps, defaultValue := vbenFormComponent(f)
		rules := vbenFormRules(f)

		item := "    {\n" +
			"      component: '" + component + "',\n" +
			"      fieldName: '" + f.Name + "',\n" +
			"      label: $t('page." + modelCamel + "." + f.Name + "'),"
		if defaultValue != nil {
			item += "\n      defaultValue: " + *defaultValue + ","
		}
		if rules != "" {
			item += "\n      rules: " + rules + ","
		}
		item += "\n      componentProps: {\n" + extraProps + "\n      },\n    },"
		schemaItems = append(schemaItems, item)
	}

	var apiImports []string
	if hasCreate {
		apiImports = append(apiImports, "useCreate"+modelPascal)
	}
	if hasUpdate {
		apiImports = append(apiImports, "useUpdate"+modelPascal)
	}
	if hasStatusEnum {
		apiImports = append(apiImports, "statusList")
	}
	// 只读服务(仅 List/Get)没有 Create/Update,此时不能留下 `import {\n  ,\n}` 的空壳:
	// 那是一行都过不了解析的语法错误。
	apiImportBlock := ""
	if len(apiImports) > 0 {
		apiImportBlock = "import {\n  " + strings.Join(apiImports, ",\n  ") + ",\n} from '#/api';\n"
	}

	_ = serviceName
	var sb strings.Builder
	sb.WriteString(`<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
` + apiImportBlock)

	if hasCreate {
		sb.WriteString("const { mutateAsync: create" + modelPascal + " } = useCreate" + modelPascal + "();\n")
	}
	if hasUpdate {
		sb.WriteString("const { mutateAsync: update" + modelPascal + " } = useUpdate" + modelPascal + "();\n")
	}

	sb.WriteString(`
const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.` + modelCamel + `.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.` + modelCamel + `.moduleName') }),
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
` + strings.Join(schemaItems, "\n") + `
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
        ? create` + modelPascal + `(values)
        : update` + modelPascal + `({ id: data.value.row.id, values }));

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
`)

	return sb.String()
}

// vbenFormComponent 推断 useVbenForm schema 的组件类型
func vbenFormComponent(field *ParsedField) (component, extraProps string, defaultValue *string) {
	inputProps := func() string {
		return "        placeholder: $t('ui.placeholder.input'),\n        allowClear: true,"
	}

	if field.IsBoolean {
		dv := "false"
		if field.Name == "isEnabled" || field.Name == "isDefault" {
			dv = "true"
		}
		return "Switch", "        class: 'w-auto',", &dv
	}
	if isStatusField(field) {
		dv := "'ON'"
		return "RadioGroup",
			"        optionType: 'button',\n" +
				"        buttonStyle: 'solid',\n" +
				"        class: 'flex flex-wrap',\n" +
				"        options: statusList,",
			&dv
	}
	if field.IsInteger && strings.Contains(strings.ToLower(field.Name), "sort") {
		dv := "1"
		return "InputNumber", inputProps(), &dv
	}
	lower := strings.ToLower(field.Name)
	if strings.Contains(lower, "description") || strings.Contains(lower, "remark") {
		return "Textarea", inputProps(), nil
	}
	return "Input", inputProps(), nil
}

// vbenFormRules 推断表单校验规则（空串 = 无规则）
func vbenFormRules(field *ParsedField) string {
	if field.IsBoolean {
		return ""
	}
	switch field.Name {
	case "sortOrder", "description", "remark":
		return ""
	}
	if isStatusField(field) {
		return "'selectRequired'"
	}
	return "'required'"
}

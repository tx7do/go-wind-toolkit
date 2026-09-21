package frontendgen

import "strings"

// ==============================
// 字段推断辅助（element）
// ==============================

// elementFormType 表单组件推断
type elementFormType struct {
	typeName  string
	component string
	attrs     string
}

func elementGetFieldFormType(field *ParsedField) elementFormType {
	lower := strings.ToLower(field.Name)
	if field.IsBoolean {
		return elementFormType{typeName: "switch", component: "ElSwitch", attrs: ""}
	}
	if field.IsEnum && len(field.EnumValues) > 0 {
		if len(field.EnumValues) <= 3 {
			return elementFormType{typeName: "radio", component: "ElRadioGroup", attrs: ""}
		}
		return elementFormType{
			typeName:  "select",
			component: "ElSelect",
			attrs:     "\n          <ElOption v-for=\"item in " + elementEnumListVar(field) + "\" :key=\"item.value\" :label=\"item.label\" :value=\"item.value\" />",
		}
	}
	if field.IsInteger && strings.Contains(lower, "sort") {
		return elementFormType{
			typeName:  "input-number",
			component: "ElInputNumber",
			attrs:     "          :min=\"1\"\n          style=\"width: 100%\"",
		}
	}
	if field.IsDate {
		return elementFormType{
			typeName:  "date-picker",
			component: "ElDatePicker",
			attrs:     "          type=\"datetime\"\n          style=\"width: 100%\"",
		}
	}
	if field.IsArray {
		return elementFormType{typeName: "input", component: "ElInput", attrs: ""}
	}
	if strings.Contains(lower, "description") || strings.Contains(lower, "remark") {
		return elementFormType{
			typeName:  "textarea",
			component: "ElInput",
			attrs:     "          type=\"textarea\"\n          :rows=\"3\"",
		}
	}
	return elementFormType{typeName: "input", component: "ElInput", attrs: ""}
}

// elementEnumListVar 枚举字段在抽屉 <script setup> 里的选项常量名。
// 模板里的 v-for 与脚本里的声明必须共用这个函数,否则又会引用一个没人声明的标识符。
func elementEnumListVar(field *ParsedField) string {
	return field.Name + "List"
}

// isElementStatusEnum 判定枚举列是否走 statusToColor/statusToName 的标签渲染。
// 这两个函数由目标工程的 @/api/composables 提供,只承载 status 语义;
// 拿去渲染 requestMethod 之类的枚举,既 import 不到也译不出名字。
func isElementStatusEnum(field *ParsedField) bool {
	return field.IsEnum && len(field.EnumValues) > 0 && field.Name == "status"
}

var elementSearchSkipFields = map[string]bool{
	"id": true, "description": true, "remark": true, "sortOrder": true,
	"isDefault": true, "isEnabled": true,
}

func elementSearchFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if !elementSearchSkipFields[f.Name] && !f.IsArray && !f.IsDate {
			out = append(out, *f)
		}
	}
	return out
}

func elementTableFields(fields []ParsedField) []ParsedField {
	var out []ParsedField
	for i := range fields {
		f := &fields[i]
		if f.Name != "id" && !f.IsArray {
			out = append(out, *f)
		}
	}
	return out
}

// ==============================
// 列表页 index.vue
// ==============================

// elementPageCode 生成 ProPage 列表页（对应 TS 版 page-template.ts generatePageCode）
func elementPageCode(service *ParsedService, serviceName, modulePath string) string {
	modelPascal := toPascalCase(service.ModelName)
	i18nModuleKey := "pages." + toCamelCase(service.ModelName)

	crudPaths := GetCrudPaths(service)
	hasList := crudPaths.List != nil
	hasDelete := crudPaths.Delete != nil
	if !hasList {
		return "// " + service.TagName + " 没有 List 操作，无法生成列表页面"
	}

	_ = serviceName
	_ = modulePath

	searchFields := elementSearchFields(service.Fields)
	tableFields := elementTableFields(service.Fields)

	// 搜索字段配置
	var searchFieldsCode []string
	for i := range searchFields {
		f := &searchFields[i]
		searchFieldsCode = append(searchFieldsCode,
			"      {\n"+
				"        type: \"input\",\n"+
				"        label: $t(\""+i18nModuleKey+"."+f.Name+"\"),\n"+
				"        field: \""+f.Name+"\",\n"+
				"        attrs: { placeholder: $t(\"common.placeholder.input\"), clearable: true },\n"+
				"      },")
	}

	// 表格列配置
	var columnCodes []string
	columnCodes = append(columnCodes, `      { type: "index", label: $t("common.table.seq"), width: 60 },`)

	for i := range tableFields {
		field := &tableFields[i]
		lower := strings.ToLower(field.Name)

		var col string
		if len(columnCodes) == 1 {
			// 第一个字段固定在左侧
			col = "      {\n" +
				"        prop: \"" + field.Name + "\",\n" +
				"        label: $t(\"" + i18nModuleKey + "." + field.Name + "\"),\n" +
				"        minWidth: 120,\n" +
				"        fixed: \"left\",\n" +
				"      },"
		} else if field.IsBoolean || isElementStatusEnum(field) {
			col = "      {\n" +
				"        prop: \"" + field.Name + "\",\n" +
				"        label: $t(\"" + i18nModuleKey + "." + field.Name + "\"),\n" +
				"        width: 100,\n" +
				"        slotName: \"" + field.Name + "\",\n" +
				"      },"
		} else if field.IsDate {
			col = "      {\n" +
				"        prop: \"" + field.Name + "\",\n" +
				"        label: $t(\"" + i18nModuleKey + "." + field.Name + "\"),\n" +
				"        minWidth: 160,\n" +
				"        cellType: \"date\",\n" +
				"        dateFormat: \"YYYY-MM-DD HH:mm:ss\",\n" +
				"      },"
		} else if field.IsInteger && strings.Contains(lower, "sort") {
			col = "      {\n" +
				"        prop: \"" + field.Name + "\",\n" +
				"        label: $t(\"common.table.sortOrder\"),\n" +
				"        width: 100,\n" +
				"        align: \"right\",\n" +
				"      },"
		} else {
			col = "      {\n" +
				"        prop: \"" + field.Name + "\",\n" +
				"        label: $t(\"" + i18nModuleKey + "." + field.Name + "\"),\n" +
				"        minWidth: 120,\n" +
				"      },"
		}
		columnCodes = append(columnCodes, col)
	}

	// 操作列
	columnCodes = append(columnCodes, `      {
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

	// 模板中的 slot 定义。slot 与 import 必须同源:模板引用了谁,脚本就得导入谁,
	// 反之导入没被用到的符号也是多余的。
	var slotCodes []string
	hasBoolSlot, hasStatusSlot := false, false
	for i := range tableFields {
		field := &tableFields[i]
		if field.IsBoolean {
			hasBoolSlot = true
			slotCodes = append(slotCodes,
				"      <!-- "+field.Description+" -->\n"+
					"      <template #"+field.Name+"=\"scope\">\n"+
					"        <ElTag size=\"small\" :type=\"scope.row."+field.Name+" ? 'success' : 'info'\" effect=\"plain\">\n"+
					"          {{ enableBoolToName(scope.row."+field.Name+") }}\n"+
					"        </ElTag>\n"+
					"      </template>")
		} else if isElementStatusEnum(field) {
			hasStatusSlot = true
			slotCodes = append(slotCodes,
				"      <!-- "+field.Description+" -->\n"+
					"      <template #"+field.Name+"=\"scope\">\n"+
					"        <ElTag size=\"small\" effect=\"dark\" round :color=\"statusToColor(scope.row."+field.Name+")\">\n"+
					"          {{ statusToName(scope.row."+field.Name+") }}\n"+
					"        </ElTag>\n"+
					"      </template>")
		}
	}

	// composable imports
	var composableImports []string
	if hasBoolSlot {
		composableImports = append(composableImports, "enableBoolToName")
	}
	if hasList {
		composableImports = append(composableImports, "fetchList"+modelPascal+"s")
	}
	if hasDelete {
		composableImports = append(composableImports, "useDelete"+modelPascal)
	}
	if hasStatusSlot {
		composableImports = append(composableImports, "statusToColor", "statusToName")
	}

	deleteLine := ""
	if hasDelete {
		deleteLine = "const { mutateAsync: delete" + modelPascal + " } = useDelete" + modelPascal + "();\n"
	}
	deleteAction := ""
	if hasDelete {
		deleteAction = "\n    deleteAction: async (ids: string) => {\n" +
			"      await delete" + modelPascal + "({ id: ids as any });\n" +
			"    },"
	}

	// ElTag 只在真的有标签 slot 时才导入
	tagImport := ""
	if hasBoolSlot || hasStatusSlot {
		tagImport = "import { ElTag } from \"element-plus\";\n"
	}

	var sb strings.Builder
	sb.WriteString(`<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
` + strings.Join(slotCodes, "\n\n") + `
    </ProPage>

    <!-- 新增/编辑抽屉 -->
` + "    \n" + `    <` + modelPascal + `Drawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
` + tagImport + `
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ` + modelPascal + `Drawer from "./` + service.KebabName + `-drawer.vue";

import {
  ` + strings.Join(composableImports, ",\n  ") + `,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
` + deleteLine + `
const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
` + strings.Join(searchFieldsCode, "\n") + `
    ],
  },

  table: {
    listAction: async (query) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchList` + modelPascal + `s(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },` + deleteAction + `
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
` + strings.Join(columnCodes, "\n") + `
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
`)

	return sb.String()
}

// ==============================
// 编辑抽屉 drawer.vue
// ==============================

// elementDrawerCode 生成 ElDrawer + ElForm 编辑抽屉
func elementDrawerCode(service *ParsedService) string {
	modelCamel := toCamelCase(service.ModelName)
	modelPascal := toPascalCase(service.ModelName)
	i18nModuleKey := "pages." + modelCamel

	crudPaths := GetCrudPaths(service)
	hasCreate := crudPaths.Create != nil
	hasUpdate := crudPaths.Update != nil

	// 表单字段（排除系统字段和ID）
	var formFields []ParsedField
	for i := range service.Fields {
		f := &service.Fields[i]
		if f.Name != "id" && !f.IsDate {
			formFields = append(formFields, *f)
		}
	}

	// 表单项模板代码
	var formItemCodes []string
	for i := range formFields {
		f := &formFields[i]
		formType := elementGetFieldFormType(f)
		label := "$t('" + i18nModuleKey + "." + f.Name + "')"
		prop := "prop=\"" + f.Name + "\""

		switch formType.typeName {
		case "switch":
			formItemCodes = append(formItemCodes,
				"      <ElFormItem :label=\""+label+"\" "+prop+">\n"+
					"        <"+formType.component+" v-model=\"formData."+f.Name+"\" />\n"+
					"      </ElFormItem>")
		case "radio":
			var radioButtons []string
			for _, v := range f.EnumValues {
				radioButtons = append(radioButtons,
					"          <ElRadioButton :value=\"'"+v+"'\">{{ $t(\"enum."+f.Name+"."+v+"\") }}</ElRadioButton>")
			}
			formItemCodes = append(formItemCodes,
				"      <ElFormItem :label=\""+label+"\" "+prop+">\n"+
					"        <ElRadioGroup v-model=\"formData."+f.Name+"\">\n"+
					strings.Join(radioButtons, "\n")+"\n"+
					"        </ElRadioGroup>\n"+
					"      </ElFormItem>")
		case "select":
			formItemCodes = append(formItemCodes,
				"      <ElFormItem :label=\""+label+"\" "+prop+">\n"+
					"        <ElSelect v-model=\"formData."+f.Name+"\" placeholder=\""+f.Description+"\">\n"+
					strings.TrimPrefix(formType.attrs, "\n")+"\n"+
					"        </ElSelect>\n"+
					"      </ElFormItem>")
		case "input-number":
			formItemCodes = append(formItemCodes,
				"      <ElFormItem :label=\""+label+"\" "+prop+">\n"+
					"        <ElInputNumber\n"+
					"          v-model=\"formData."+f.Name+"\"\n"+
					formType.attrs+"\n"+
					"          :placeholder=\"$t('common.placeholder.input')\"\n"+
					"        />\n"+
					"      </ElFormItem>")
		default:
			attrsLine := ""
			if formType.attrs != "" {
				attrsLine = formType.attrs + "\n"
			}
			clearable := ""
			if formType.typeName != "textarea" {
				clearable = "clearable"
			}
			formItemCodes = append(formItemCodes,
				"      <ElFormItem :label=\""+label+"\" "+prop+">\n"+
					"        <"+formType.component+"\n"+
					"          v-model=\"formData."+f.Name+"\"\n"+
					attrsLine+
					"          :placeholder=\"$t('common.placeholder.input')\"\n"+
					"          "+clearable+"\n"+
					"        />\n"+
					"      </ElFormItem>")
		}
	}

	// 枚举选项常量:elementGetFieldFormType 只写下 `<ElOption v-for="item in xList">` 的引用,
	// 声明必须在这里补上,否则生成的抽屉一渲染就 ReferenceError。
	// label 直接用 enum 原值,不依赖工程里未必存在的 enum.* 词条。
	var enumListDecls []string
	for i := range formFields {
		f := &formFields[i]
		if elementGetFieldFormType(f).typeName != "select" {
			continue
		}
		items := make([]string, 0, len(f.EnumValues))
		for _, v := range f.EnumValues {
			items = append(items, "  { label: \""+v+"\", value: \""+v+"\" },")
		}
		enumListDecls = append(enumListDecls,
			"// "+f.Description+" 选项(取自 OpenAPI enum)\n"+
				"const "+elementEnumListVar(f)+" = [\n"+strings.Join(items, "\n")+"\n];")
	}
	enumDeclBlock := ""
	if len(enumListDecls) > 0 {
		enumDeclBlock = strings.Join(enumListDecls, "\n\n") + "\n\n"
	}

	// formData 字段默认值
	var formDataDefaults []string
	for i := range formFields {
		f := &formFields[i]
		switch {
		case f.IsBoolean:
			dv := "false"
			if f.Name == "isEnabled" || f.Name == "isDefault" {
				dv = "true"
			}
			formDataDefaults = append(formDataDefaults, "  "+f.Name+": "+dv+",")
		case f.IsInteger:
			formDataDefaults = append(formDataDefaults, "  "+f.Name+": 1,")
		case f.IsArray:
			formDataDefaults = append(formDataDefaults, "  "+f.Name+": [] as any[],")
		default:
			formDataDefaults = append(formDataDefaults, "  "+f.Name+": \"\",")
		}
	}

	// formRules
	var formRulesCode []string
	for i := range formFields {
		f := &formFields[i]
		if f.IsBoolean || f.Name == "sortOrder" || f.Name == "description" || f.Name == "remark" {
			continue
		}
		formRulesCode = append(formRulesCode,
			"  "+f.Name+": [{ required: true, message: $t(\"common.validation.required\"), trigger: \"blur\" }],")
	}

	// resetForm
	var resetFormCode []string
	for i := range formFields {
		f := &formFields[i]
		switch {
		case f.IsBoolean:
			dv := "false"
			if f.Name == "isEnabled" || f.Name == "isDefault" {
				dv = "true"
			}
			resetFormCode = append(resetFormCode, "  formData."+f.Name+" = "+dv+";")
		case f.IsInteger:
			resetFormCode = append(resetFormCode, "  formData."+f.Name+" = 1;")
		case f.IsArray:
			resetFormCode = append(resetFormCode, "  formData."+f.Name+" = [];")
		default:
			resetFormCode = append(resetFormCode, "  formData."+f.Name+" = \"\";")
		}
	}

	// composable imports
	var composableImports []string
	if hasCreate {
		composableImports = append(composableImports, "useCreate"+modelPascal)
	}
	if hasUpdate {
		composableImports = append(composableImports, "useUpdate"+modelPascal)
	}

	createLine := ""
	if hasCreate {
		createLine = "const { mutateAsync: create" + modelPascal + " } = useCreate" + modelPascal + "();"
	}
	updateLine := ""
	if hasUpdate {
		updateLine = "const { mutateAsync: update" + modelPascal + " } = useUpdate" + modelPascal + "();"
	}

	// 只读服务(仅 List/Get)既没有 Create 也没有 Update,这时整条 composable import
	// 必须省略——留着 `import {\n  ,\n}` 的壳是一行都过不了解析的语法错误。
	composableImportBlock := ""
	if len(composableImports) > 0 {
		composableImportBlock = "import {\n  " + strings.Join(composableImports, ",\n  ") + ",\n} from \"@/api/composables\";\n"
	}

	// handleSubmit 分支
	var submitBranch string
	if hasCreate {
		submitBranch += "\n    if (isCreate.value) {\n" +
			"      await create" + modelPascal + "(values);\n" +
			"      ElMessage.success($t(\"common.notification.createSuccess\"));\n" +
			"    }"
	}
	if hasUpdate {
		sep := ""
		if hasCreate {
			sep = " else"
		}
		submitBranch += sep + " {\n" +
			"      await update" + modelPascal + "({ id: currentId.value!, values });\n" +
			"      ElMessage.success($t(\"common.notification.updateSuccess\"));\n" +
			"    }"
	}

	var sb strings.Builder
	sb.WriteString(`<template>
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

` + strings.Join(formItemCodes, "\n\n") + `
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

` + composableImportBlock + `import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{
  success: [];
}>();
` + createLine + "\n" + updateLine + `

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

` + enumDeclBlock + `// 表单数据
const formData = reactive({
` + strings.Join(formDataDefaults, "\n") + `
});

// 表单验证规则
const formRules = {
` + strings.Join(formRulesCode, "\n") + `
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("` + i18nModuleKey + `.moduleName") })
    : $t("common.modal.update", { moduleName: $t("` + i18nModuleKey + `.moduleName") })
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
` + strings.Join(resetFormCode, "\n") + `

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };
` + submitBranch + `

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
`)

	return sb.String()
}

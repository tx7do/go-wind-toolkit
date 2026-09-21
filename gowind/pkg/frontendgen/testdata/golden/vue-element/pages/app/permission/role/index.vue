<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 受保护角色 -->
      <template #isProtected="scope">
        <ElTag size="small" :type="scope.row.isProtected ? 'success' : 'info'" effect="plain">
          {{ enableBoolToName(scope.row.isProtected) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    
    <RoleDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import RoleDrawer from "./role-drawer.vue";

import {
  enableBoolToName,
  fetchListRoles,
  useDeleteRole,
  statusToColor,
  statusToName,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
const { mutateAsync: deleteRole } = useDeleteRole();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.role.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.role.isProtected"),
        field: "isProtected",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.role.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.role.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.role.tenantName"),
        field: "tenantName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListRoles(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteRole({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      {
        prop: "code",
        label: $t("pages.role.code"),
        minWidth: 120,
        fixed: "left",
      },
      {
        prop: "description",
        label: $t("pages.role.description"),
        minWidth: 120,
      },
      {
        prop: "isProtected",
        label: $t("pages.role.isProtected"),
        width: 100,
        slotName: "isProtected",
      },
      {
        prop: "name",
        label: $t("pages.role.name"),
        minWidth: 120,
      },
      {
        prop: "sortOrder",
        label: $t("common.table.sortOrder"),
        width: 100,
        align: "right",
      },
      {
        prop: "status",
        label: $t("pages.role.status"),
        width: 100,
        slotName: "status",
      },
      {
        prop: "tenantName",
        label: $t("pages.role.tenantName"),
        minWidth: 120,
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "delete", label: $t("common.button.delete"), icon: "lucide:trash-2", attrs: { type: "danger" } },
        ],
      },
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

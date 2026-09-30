<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 是否成功 -->
      <template #isSuccess="scope">
        <ElTag size="small" :type="scope.row.isSuccess ? 'success' : 'info'" effect="plain">
          {{ enableBoolToName(scope.row.isSuccess) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    
    <ApiAuditLogDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ApiAuditLogDrawer from "./api-audit-log-drawer.vue";

import {
  enableBoolToName,
  fetchListApiAuditLogs,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.apiAuditLog.duration"),
        field: "duration",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.apiAuditLog.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.apiAuditLog.isSuccess"),
        field: "isSuccess",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.apiAuditLog.operatorName"),
        field: "operatorName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.apiAuditLog.requestMethod"),
        field: "requestMethod",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.apiAuditLog.requestPath"),
        field: "requestPath",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListApiAuditLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      {
        prop: "duration",
        label: $t("pages.apiAuditLog.duration"),
        minWidth: 120,
        fixed: "left",
      },
      {
        prop: "ipAddress",
        label: $t("pages.apiAuditLog.ipAddress"),
        minWidth: 120,
      },
      {
        prop: "isSuccess",
        label: $t("pages.apiAuditLog.isSuccess"),
        width: 100,
        slotName: "isSuccess",
      },
      {
        prop: "operatedAt",
        label: $t("pages.apiAuditLog.operatedAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operatorName",
        label: $t("pages.apiAuditLog.operatorName"),
        minWidth: 120,
      },
      {
        prop: "requestMethod",
        label: $t("pages.apiAuditLog.requestMethod"),
        minWidth: 120,
      },
      {
        prop: "requestPath",
        label: $t("pages.apiAuditLog.requestPath"),
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

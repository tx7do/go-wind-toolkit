<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">

    </ProPage>

    <!-- 新增/编辑抽屉 -->
    
    <OrgUnitDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import OrgUnitDrawer from "./org-unit-drawer.vue";

import {
  fetchListOrgUnits,
  useDeleteOrgUnit,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
const { mutateAsync: deleteOrgUnit } = useDeleteOrgUnit();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.orgUnit.leader"),
        field: "leader",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.orgUnit.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.orgUnit.parentId"),
        field: "parentId",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListOrgUnits(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteOrgUnit({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      {
        prop: "leader",
        label: $t("pages.orgUnit.leader"),
        minWidth: 120,
        fixed: "left",
      },
      {
        prop: "name",
        label: $t("pages.orgUnit.name"),
        minWidth: 120,
      },
      {
        prop: "parentId",
        label: $t("pages.orgUnit.parentId"),
        minWidth: 120,
      },
      {
        prop: "sortOrder",
        label: $t("common.table.sortOrder"),
        width: 100,
        align: "right",
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

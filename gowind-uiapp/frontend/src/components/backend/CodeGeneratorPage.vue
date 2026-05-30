<script setup lang="ts">
import {ref, reactive, onMounted} from 'vue'

import {
  EditGeneratorOption,
  GetGeneratorOptions,
  GetProjectInfo,
  SetGeneratorOption,
  OpenProject,
  SelectFolder,
} from "../../../wailsjs/go/main/App";
import {generator, detect} from "../../../wailsjs/go/models";
import {EventsOn} from "../../../wailsjs/runtime";

import DatabaseImporterModal from "./DatabaseImporterModal.vue";
import SqlImporterModal from "./SqlImporterModal.vue";
import GRPCCodeGenerateModal from "./GRPCCodeGenerateModal.vue";
import RESTCodeGenerateModal from "./RESTCodeGenerateModal.vue";

const openDatabaseImporter = ref<boolean>(false);
const openSqlImporter = ref<boolean>(false);

const grpcCodeGenerateImporter = ref<boolean>(false);
const restCodeGenerateImporter = ref<boolean>(false);

// 快速选择服务
const quickSelectService = ref<string>('');

// 项目信息
const projectInfo = ref<detect.ProjectInfo>()

// 打开后端项目
async function handleOpenProject() {
  try {
    const path = await SelectFolder();
    if (path) {
      const pi = await OpenProject(path);
      projectInfo.value = pi;
      await refreshServiceOptions();
    }
  } catch (err) {
    console.error('选择文件夹出错：', err);
  }
}

// 表格数据
const tableData = ref<Array<{ id: number; tableName: string; service: string; exclude: boolean }>>([])

// 服务选项
const serviceOptions = reactive<Array<{ label: string; value: string }>>([])

function handleGenerateGRPCCode() {
  grpcCodeGenerateImporter.value = true;
}

function handleGenerateRESTCode() {
  restCodeGenerateImporter.value = true;
}

// ==================== 操作 ====================

function handleDatabaseImport() {
  openDatabaseImporter.value = true;
}

function handleSQLImport() {
  openSqlImporter.value = true;
}

// 一键选中所有表的某个服务
async function handleQuickSelectService(service: string) {
  tableData.value.forEach(row => {
    row.service = service;
  });

  const opts = await GetGeneratorOptions();
  for (let i = 0; i < opts.length; i++) {
    opts[i].service = service;
  }
  await SetGeneratorOption(opts);

  quickSelectService.value = '';
}

/**
 * 处理服务选择变更
 */
async function handleServiceChange(row: generator.Option) {
  console.log('服务已更改：', row);
  await EditGeneratorOption(row);

  const opts = await GetGeneratorOptions();
  console.log('切换服务：', opts);
}

/**
 * 处理排除状态变更
 */
async function handleExcludeChange(row: generator.Option) {
  console.log('排除状态已更改：', row);
  await EditGeneratorOption(row);

  const opts = await GetGeneratorOptions();
  console.log('切换排除状态：', opts);
}

/**
 * 刷新服务选项列表
 */
async function refreshServiceOptions() {
  const pi = await GetProjectInfo();
  if (pi && pi.Services) {
    serviceOptions.length = 0; // 清空现有选项
    pi.Services.forEach(service => {
      serviceOptions.push({label: service, value: service});
    });
  }
}

async function refreshTableData() {
  const opts = await GetGeneratorOptions();
  console.log('刷新表数据，当前选项：', opts);

  tableData.value = []; // 清空现有数据
  // if (!opts || !opts.Tables) {
  //   return;
  // }

  tableData.value = opts;

  console.log('刷新表数据完成：', tableData);
}

EventsOn('project-opened', () => {
  console.log("project-opened");
  refreshServiceOptions();
  // 同时刷新项目信息
  GetProjectInfo().then(pi => {
    if (pi) projectInfo.value = pi;
  });
})
EventsOn('table-imported', () => {
  console.log("table-imported");
  refreshTableData();
})
</script>

<template>
  <div class="code-generator-container">
    <!-- 项目信息栏 -->
    <div class="project-bar">
      <div class="project-bar-left">
        <a-button type="primary" @click="handleOpenProject">打开项目</a-button>
      </div>
      <div class="project-bar-right" v-if="projectInfo">
        <div class="info-item">
          <span class="info-label">项目</span>
          <span class="info-value">{{ projectInfo.ModPath }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Go版本</span>
          <span class="info-value">{{ projectInfo.GoVersion }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">服务数</span>
          <span class="info-value">{{ projectInfo.Services?.length ?? 0 }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">有API</span>
          <span class="info-value">{{ projectInfo.HasApi ? '有' : '无' }}</span>
        </div>
      </div>
      <div class="project-bar-right" v-else>
        <span class="prompt-text">请先打开一个微服务项目</span>
      </div>
    </div>

    <a-card title="后端代码生成" class="full-card">
      <template #extra>
        <a-space>
          <a-button type="primary" @click="handleDatabaseImport">数据库导入</a-button>
          <a-button type="primary" @click="handleSQLImport">SQL导入</a-button>
          <a-button type="primary" danger @click="handleGenerateGRPCCode">生成gRPC服务</a-button>
          <a-button type="primary" danger @click="handleGenerateRESTCode">生成BFF服务</a-button>
        </a-space>
      </template>

      <vxe-table
          :data="tableData"
          :row-config="{ keyField: 'id' }"
          size="small"
          class="table-content"
      >
        <vxe-column field="tableName" title="表名" width="40%"/>
        <vxe-column field="service" title="服务" width="30%">
          <template #header>
            <div class="service-header">
              <span>服务</span>
              <a-select
                  v-model:value="quickSelectService"
                  :options="serviceOptions"
                  placeholder="一键全选"
                  style="width: 150px; margin-left: 8px"
                  @change="handleQuickSelectService"
                  allow-clear
              />
            </div>
          </template>
          <template #default="{ row }">
            <a-select
                v-model:value="row.service"
                :options="serviceOptions"
                placeholder="选择服务"
                style="width: 100%"
                @change="handleServiceChange(row)"
            />
          </template>
        </vxe-column>
        <vxe-column field="exclude" title="排除" width="30%" align="center">
          <template #default="{ row }">
            <a-switch
                v-model:checked="row.exclude"
                :style="{ backgroundColor: row.exclude ? '#ff4d4f' : undefined }"
                @change="handleExcludeChange(row)"
            />
          </template>
        </vxe-column>
      </vxe-table>
    </a-card>
  </div>
  <DatabaseImporterModal
      v-model:open="openDatabaseImporter"
  />
  <SqlImporterModal
      v-model:open="openSqlImporter"/>
  <GRPCCodeGenerateModal
      v-model:open="grpcCodeGenerateImporter"/>
  <RESTCodeGenerateModal
      v-model:open="restCodeGenerateImporter"/>
</template>

<style scoped>
.code-generator-container {
  width: 100%;
  height: 100%;
  padding: 0;
  margin: 0;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

/* 项目信息栏 */
.project-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  margin-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.project-bar-left {
  display: flex;
  align-items: center;
}

.project-bar-right {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 13px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-label {
  color: #8c8c8c;
  font-size: 12px;
}

.info-value {
  color: #262626;
  font-weight: 500;
}

.prompt-text {
  color: #8c8c8c;
  font-size: 13px;
  font-style: italic;
}

.full-card {
  width: 100%;
  flex: 1;
  box-sizing: border-box;
}

:deep(.ant-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

:deep(.ant-card-head) {
  flex-shrink: 0;
}

:deep(.ant-card-body) {
  flex: 1;
  overflow: auto;
  padding: 16px;
  box-sizing: border-box;
}

/* Switch 排除状态样式 */
:deep(.ant-switch-checked) {
  background-color: #ff4d4f !important;
}

.table-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background-color: #fafafa;
  border-radius: 4px;
}

.table-header span {
  font-weight: 500;
  color: #333;
}

.service-header {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
}
</style>

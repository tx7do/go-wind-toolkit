<script setup lang="ts">
import {computed, ref} from 'vue'
import {message} from 'ant-design-vue'
import {useI18n} from 'vue-i18n'
import {EditOutlined, PlusOutlined, DeleteOutlined, TableOutlined} from '@ant-design/icons-vue'
import {useAI} from '../../../stores/ai'

const {t} = useI18n()
const {
  ddlContent, partitions, partitionWarnings, partitionLoading, hasPartitions,
  editingPartitions, unassignedTables,
  partition, beginEditPartitions, cancelEditPartitions, savePartitions,
  removeTable, moveTable, assignTable, addService, removeService, cancelTask,
} = useAI()

const saving = ref(false)

// 服务名展示:编辑中新服务可能为空,菜单里给占位文案
function serviceNameOrPlaceholder(idx: number): string {
  const name = partitions.value[idx]?.serviceName?.trim()
  return name || t('ai.partition.unnamed')
}

async function handlePartition() {
  if (!ddlContent.value.trim()) {
    message.warning(t('ai.partition.ddlRequired'))
    return
  }
  const result = await partition()
  if (result?.success) {
    message.success(t('ai.partition.success'))
  } else {
    message.error(result?.error || t('ai.partition.failed'))
  }
}

async function handleBeginEdit() {
  const err = await beginEditPartitions()
  if (err !== '') {
    message.error(err)
  }
}

// 保存前先做名字校验(用 t() 出提示),再交 store 持久化
async function handleSave() {
  const names = new Set<string>()
  for (const p of partitions.value) {
    const name = p.serviceName?.trim() || ''
    if (!name) {
      message.warning(t('ai.partition.emptyName'))
      return
    }
    if (names.has(name)) {
      message.warning(t('ai.partition.dupName', {name}))
      return
    }
    names.add(name)
  }
  saving.value = true
  try {
    const err = await savePartitions()
    if (err === '') {
      message.success(t('ai.partition.saved'))
    } else {
      message.error(err)
    }
  } finally {
    saving.value = false
  }
}

// 菜单点击处理器用闭包工厂:Vue 模板表达式不支持内联类型注解,
// 类型留在 script 侧,模板里只调 assignHandler(table) / moveHandler(table, idx)。
function assignHandler(table: string) {
  return (info: { key: string | number }) => assignTable(table, Number(info.key))
}

function moveHandler(table: string, fromIdx: number) {
  return (info: { key: string | number }) => moveTable(table, fromIdx, Number(info.key))
}

// 编辑态下服务卡片的表格列,给 template 的 v-for 用
const serviceIndexes = computed(() => partitions.value.map((_, i) => i))
</script>

<template>
  <a-card size="small">
    <template #title>
      <div class="card-title-row">
        <span>{{ t('ai.partition.title') }}</span>
        <div class="card-actions">
          <!-- 视图态才有"编辑";生成按钮在编辑态禁用,避免编辑中途被 AI 重跑覆盖 -->
          <a-button v-if="hasPartitions && !editingPartitions" @click="handleBeginEdit">
            <EditOutlined style="margin-right: 4px"/> {{ t('ai.partition.edit') }}
          </a-button>
          <a-button v-if="partitionLoading" danger @click="cancelTask('partition')">
            {{ t('ai.cancel') }}
          </a-button>
          <a-button type="primary" :loading="partitionLoading" :disabled="editingPartitions" @click="handlePartition">
            <TableOutlined style="margin-right: 4px"/> {{ t('ai.partition.generateBtn') }}
          </a-button>
        </div>
      </div>
    </template>

    <!-- 幻觉表名剔除/跨服务去重等清洗告警 -->
    <a-alert
        v-if="partitionWarnings.length > 0"
        type="warning"
        show-icon
        style="margin-bottom: 8px"
    >
      <template #message>{{ t('ai.partition.warnings') }}</template>
      <template #description>
        <ul class="warning-list">
          <li v-for="(w, idx) in partitionWarnings" :key="idx">{{ w }}</li>
        </ul>
      </template>
    </a-alert>

    <a-alert
        v-if="editingPartitions"
        :message="t('ai.partition.editing')"
        type="info"
        show-icon
        style="margin-bottom: 8px"
    />

    <template v-if="hasPartitions">
      <!-- 未分配池:编辑态展示,池表可分配给任意服务 -->
      <a-card v-if="editingPartitions" size="small" class="partition-card unassigned-card">
        <template #title>
          <span class="unassigned-title">{{ t('ai.partition.unassigned') }} ({{ unassignedTables.length }})</span>
        </template>
        <div v-if="unassignedTables.length > 0" class="partition-tables">
          <a-dropdown v-for="table in unassignedTables" :key="table" :trigger="['click']">
            <a-tag color="orange" class="pool-tag">{{ table }}</a-tag>
            <template #overlay>
              <a-menu @click="assignHandler(table)">
                <a-menu-item v-for="j in serviceIndexes" :key="j">
                  {{ serviceNameOrPlaceholder(j) }}
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
        <span v-else class="all-assigned">{{ t('ai.partition.allAssigned') }}</span>
      </a-card>

      <div class="partition-list">
        <a-card v-for="(p, idx) in partitions" :key="idx" size="small" class="partition-card">
          <template #title>
            <div class="card-title-row">
              <!-- 编辑态:服务名可改;视图态:纯展示 -->
              <a-input
                  v-if="editingPartitions"
                  v-model:value="p.serviceName"
                  :placeholder="t('ai.partition.unnamed')"
                  size="small"
                  class="name-input"
              />
              <span v-else class="partition-name">{{ p.serviceName }}</span>
              <a-popconfirm
                  v-if="editingPartitions"
                  :title="t('ai.partition.removeConfirm')"
                  @confirm="removeService(idx)"
              >
                <a-button size="small" danger text>
                  <DeleteOutlined/>
                </a-button>
              </a-popconfirm>
            </div>
          </template>

          <!-- 编辑态:描述可改;视图态:纯展示 -->
          <a-input
              v-if="editingPartitions"
              v-model:value="p.description"
              :placeholder="t('ai.partition.descPlaceholder')"
              size="small"
              style="margin-bottom: 8px"
          />
          <p v-else class="partition-desc">{{ p.description }}</p>

          <div class="partition-tables">
            <template v-if="editingPartitions">
              <span v-if="p.tables.length === 0" class="no-tables">{{ t('ai.partition.noTables') }}</span>
              <span v-for="table in p.tables" :key="table" class="table-tag-group">
                <a-tag closable @close="removeTable(idx, table)">{{ table }}</a-tag>
                <a-dropdown :trigger="['click']">
                  <span class="move-trigger">⋯</span>
                  <template #overlay>
                    <a-menu @click="moveHandler(table, idx)">
                      <a-menu-item v-for="j in serviceIndexes" :key="j" :disabled="j === idx">
                        {{ t('ai.partition.moveTo') }} {{ serviceNameOrPlaceholder(j) }}
                      </a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </span>
            </template>
            <template v-else>
              <a-tag v-for="table in p.tables" :key="table" color="blue">{{ table }}</a-tag>
            </template>
          </div>
        </a-card>
      </div>

      <!-- 编辑态底部操作 -->
      <div v-if="editingPartitions" class="edit-footer">
        <a-button @click="addService">
          <PlusOutlined style="margin-right: 4px"/> {{ t('ai.partition.addService') }}
        </a-button>
        <div style="flex: 1"></div>
        <a-button @click="cancelEditPartitions">{{ t('ai.partition.cancel') }}</a-button>
        <a-button type="primary" :loading="saving" @click="handleSave">{{ t('ai.partition.save') }}</a-button>
      </div>
    </template>
    <a-empty v-else-if="!editingPartitions" :description="t('ai.partition.empty')"/>
  </a-card>
</template>

<style scoped>
.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 8px;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.warning-list {
  margin: 0;
  padding-left: 18px;
}

.name-input {
  max-width: 260px;
}

.partition-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.unassigned-card {
  margin-bottom: 8px;
}

.unassigned-title {
  font-weight: 600;
  color: #fa8c16;
  font-size: 13px;
}

.all-assigned {
  font-size: 12px;
  color: #8c8c8c;
}

.partition-card {
  background: #fafafa;
}

.partition-name {
  font-weight: 600;
  color: #1890ff;
  font-size: 14px;
}

.partition-desc {
  font-size: 13px;
  color: #8c8c8c;
  margin-bottom: 8px;
}

.partition-tables {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.pool-tag {
  cursor: pointer;
}

.table-tag-group {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.move-trigger {
  cursor: pointer;
  color: #8c8c8c;
  padding: 0 4px;
  font-size: 12px;
}

.move-trigger:hover {
  color: #1890ff;
}

.no-tables {
  font-size: 12px;
  color: #bfbfbf;
}

.edit-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}
</style>

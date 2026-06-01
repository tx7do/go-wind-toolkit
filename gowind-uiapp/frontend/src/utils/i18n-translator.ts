/**
 * 通用国际化翻译器
 * 从 OpenAPI 文档中提取国际化文本键值的通用逻辑
 *
 * 提供：
 * - 中文到英文的翻译映射表
 * - 基于映射表 + 字段名推断的翻译函数
 * - 从 ParsedService 中提取字段描述的工具函数
 */
import type { ParsedService, ParsedField } from './openapi-parser'
import { toPascalCase, toSnakeCase } from './case-convert'

// ==============================
// 中文描述 -> 英文翻译映射表
// ==============================

/**
 * 中文描述到英文的翻译映射表
 * 适用于所有前端框架的国际化代码生成
 */
export const zhToEnMap: Record<string, string> = {
  // 通用
  'ID': 'ID',
  '名称': 'Name',
  '编码': 'Code',
  '描述': 'Description',
  '状态': 'Status',
  '排序': 'Sort Order',
  '备注': 'Remark',
  '类型': 'Type',
  '标题': 'Title',
  '内容': 'Content',
  '创建时间': 'Created At',
  '更新时间': 'Updated At',
  '删除时间': 'Deleted At',
  '创建者': 'Created By',
  '更新者': 'Updated By',

  // 用户相关
  '用户名': 'Username',
  '昵称': 'Nickname',
  '真实姓名': 'Real Name',
  '头像': 'Avatar',
  '邮箱': 'Email',
  '手机号': 'Mobile',
  '手机号码': 'Mobile',
  '座机号': 'Telephone',
  '性别': 'Gender',
  '住址': 'Address',
  '个人描述': 'Personal Description',
  '最后登录时间': 'Last Login At',
  '最后登录IP': 'Last Login IP',
  '锁定截止时间': 'Locked Until',
  '密码': 'Password',

  // 组织相关
  '组织名称': 'Organization Name',
  '组织ID': 'Organization ID',
  '职位名称': 'Position Name',
  '父级组织': 'Parent Organization',
  '管理员': 'Admin',
  '负责人': 'Leader',

  // 权限相关
  '角色名称': 'Role Name',
  '角色编码': 'Role Code',
  '角色标识码': 'Role Code',
  '角色类型': 'Role Type',
  '受保护角色': 'Is Protected',
  '权限点': 'Permission Point',
  '权限配置': 'Permissions',
  '所属租户': 'Tenant',
  '权限名称': 'Permission Name',
  '权限编码': 'Permission Code',
  '唯一编码': 'Unique Code',

  // 字典相关
  '类型名称': 'Type Name',
  '类型编码': 'Type Code',
  '标签': 'Label',
  '值': 'Value',
  '数值': 'Numeric Value',
  '多语言配置': 'I18n Configuration',
  '语言代码': 'Language Code',
  '语言名称': 'Language Name',

  // 语言
  '本地名称': 'Native Name',
  '是否启用': 'Is Enabled',
  '是否默认': 'Is Default',

  // 文件相关
  '文件名': 'File Name',
  '文件大小': 'File Size',
  '文件类型': 'File Type',
  '存储路径': 'Storage Path',
  'MIME类型': 'MIME Type',

  // 租户相关
  '租户名称': 'Tenant Name',
  '租户ID': 'Tenant ID',
  '租户编码': 'Tenant Code',
  '联系人': 'Contact',
  '联系电话': 'Contact Phone',
  '联系邮箱': 'Contact Email',
  '域名': 'Domain',
  '有效期': 'Expired At',
  '成员数量': 'Member Count',
  '订阅时间': 'Subscription At',
  '订阅套餐': 'Subscription Plan',
  '审核状态': 'Audit Status',

  // 菜单相关
  '菜单名称': 'Menu Name',
  '菜单路径': 'Menu Path',
  '菜单图标': 'Menu Icon',
  '组件路径': 'Component Path',
  '重定向': 'Redirect',
  '是否外链': 'Is External',
  '是否缓存': 'Is KeepAlive',
  '是否可见': 'Is Visible',
  '父级菜单': 'Parent Menu',

  // 日志相关
  '请求方法': 'Request Method',
  '请求路径': 'Request Path',
  '请求参数': 'Request Params',
  '响应状态码': 'Response Status Code',
  '响应内容': 'Response Body',
  'IP地址': 'IP Address',
  'User-Agent': 'User-Agent',
  '耗时': 'Duration',
  '是否成功': 'Is Success',
  '操作者': 'Operator',

  // 任务相关
  '任务类型': 'Task Type',
  '任务数据': 'Task Payload',
  'Cron表达式': 'Cron Spec',
  '启用': 'Enable',
  '禁用': 'Inactive',

  // 策略相关
  '策略名称': 'Policy Name',
  '最大登录尝试': 'Max Login Attempts',
  '锁定时长(分钟)': 'Lock Duration (min)',
  '密码最小长度': 'Min Password Length',
  '密码过期天数': 'Password Expire Days',

  // 消息相关
  '消息分类': 'Category',
  '消息主题': 'Subject',
  '发送者': 'Sender',
  '接收者': 'Receiver',
  '已读': 'Is Read',
  '发送时间': 'Sent At',

  // 其他
  '是': 'Yes',
  '否': 'No',
  '序号': '#',
  '图标': 'Icon',
  '路径': 'Path',
  '方法': 'Method',
  '分类名称': 'Category Name',
  '分类编码': 'Category Code',
  '消息内容': 'Content',
  '消息类型': 'Type',
  '消息状态': 'Status',
  '排序值': 'Sort Order',
  '角色值': 'Role Code',
  '角色描述': 'Role Description',
  '电子邮箱': 'Email',
}

// ==============================
// 翻译函数
// ==============================

/**
 * 将中文字段描述翻译为英文
 *
 * 翻译优先级：
 * 1. 精确匹配 zhToEnMap
 * 2. 基于 fieldName 后缀/前缀/包含的关键词推断
 * 3. 默认返回 PascalCase(fieldName)
 */
export function translateToEn(zhText: string, fieldName: string): string {
  // 精确匹配
  if (zhToEnMap[zhText]) return zhToEnMap[zhText]

  // 基于 fieldName 推断
  const lower = fieldName.toLowerCase()

  if (lower.endsWith('name')) return toPascalCase(fieldName.replace(/name$/i, '')) + ' Name'
  if (lower.endsWith('code')) return toPascalCase(fieldName.replace(/code$/i, '')) + ' Code'
  if (lower.endsWith('id')) return toPascalCase(fieldName.replace(/id$/i, '')) + ' ID'
  if (lower.endsWith('type')) return toPascalCase(fieldName.replace(/type$/i, '')) + ' Type'
  if (lower.endsWith('at') || lower.endsWith('time')) return toPascalCase(fieldName) + ' Time'
  if (lower.startsWith('is')) return toPascalCase(fieldName)
  if (lower.startsWith('has')) return toPascalCase(fieldName)
  if (lower.includes('count') || lower.includes('num')) return toPascalCase(fieldName) + ' Count'
  if (lower.includes('sort')) return 'Sort Order'
  if (lower.includes('status')) return 'Status'
  if (lower.includes('desc')) return 'Description'
  if (lower.includes('remark')) return 'Remark'
  if (lower.includes('url')) return toPascalCase(fieldName)
  if (lower.includes('path')) return toPascalCase(fieldName)

  return toPascalCase(fieldName)
}

// ==============================
// 字段条目构建
// ==============================

/** 字段条目构建选项 */
export interface FieldEntriesOptions {
  /** 要跳过的字段名列表（默认 ['id']） */
  skipFields?: string[]
  /** 是否生成 Placeholder 条目（默认 false） */
  withPlaceholder?: boolean
  /** 是否生成 Required 校验提示条目（默认 false） */
  withRequired?: boolean
  /** Required 跳过的字段名（默认 ['sortOrder', 'description', 'remark']） */
  skipRequiredFields?: string[]
}

/**
 * 构建中文字段条目
 * @param fields ParsedField 列表
 * @param options 构建选项
 * @returns 键值对，如 { role_name: '角色名称', role_name_ph: '请输入角色名称', role_name_req: '请输入角色名称' }
 */
export function buildFieldEntriesZhCN(fields: ParsedField[], options?: FieldEntriesOptions): Record<string, any> {
  const skipSet = new Set(options?.skipFields ?? ['id'])
  const skipRequiredSet = new Set(options?.skipRequiredFields ?? ['sortOrder', 'description', 'remark'])
  const withPlaceholder = options?.withPlaceholder ?? false
  const withRequired = options?.withRequired ?? false

  const entries: Record<string, any> = {}
  for (const field of fields) {
    if (skipSet.has(field.name)) continue
    const key = toSnakeCase(field.name)
    const desc = field.description || field.name
    entries[key] = desc
    if (withPlaceholder) {
      entries[`${key}_ph`] = `请输入${desc}`
    }
    if (withRequired && !field.isBoolean && !skipRequiredSet.has(field.name)) {
      entries[`${key}_req`] = `请输入${desc}`
    }
  }
  return entries
}

/**
 * 构建英文字段条目
 * @param fields ParsedField 列表
 * @param options 构建选项
 * @returns 键值对，如 { role_name: 'Role Name', role_name_ph: 'Enter role name', role_name_req: 'Please enter role name' }
 */
export function buildFieldEntriesEnUS(fields: ParsedField[], options?: FieldEntriesOptions): Record<string, any> {
  const skipSet = new Set(options?.skipFields ?? ['id'])
  const skipRequiredSet = new Set(options?.skipRequiredFields ?? ['sortOrder', 'description', 'remark'])
  const withPlaceholder = options?.withPlaceholder ?? false
  const withRequired = options?.withRequired ?? false

  const entries: Record<string, any> = {}
  for (const field of fields) {
    if (skipSet.has(field.name)) continue
    const key = toSnakeCase(field.name)
    const enName = translateToEn(field.description, field.name)
    entries[key] = enName
    if (withPlaceholder) {
      entries[`${key}_ph`] = `Enter ${enName.toLowerCase()}`
    }
    if (withRequired && !field.isBoolean && !skipRequiredSet.has(field.name)) {
      entries[`${key}_req`] = `Please enter ${enName.toLowerCase()}`
    }
  }
  return entries
}

// ==============================
// 操作按钮和消息
// ==============================

/**
 * 生成通用的中文操作消息（按钮、确认、成功/失败提示等）
 * @param moduleDesc 模块中文名，如 "角色"
 */
export function buildActionMessagesZhCN(moduleDesc: string): Record<string, string> {
  return {
    action: '操作',
    create: `新建${moduleDesc}`,
    edit: `编辑${moduleDesc}`,
    yes: '是',
    no: '否',
    deleteConfirmTitle: '确认删除',
    deleteConfirmDesc: '确定要删除该{{moduleName}}吗？',
    createSuccess: '创建成功',
    updateSuccess: '更新成功',
    deleteSuccess: '删除成功',
    createFailed: '创建失败',
    updateFailed: '更新失败',
    deleteFailed: '删除失败',
    fetchFailed: '获取数据失败',
  }
}

/**
 * 生成通用的英文操作消息（按钮、确认、成功/失败提示等）
 * @param modelName 模型英文名，如 "Role"
 */
export function buildActionMessagesEnUS(modelName: string): Record<string, string> {
  return {
    action: 'Actions',
    create: `New ${modelName}`,
    edit: `Edit ${modelName}`,
    yes: 'Yes',
    no: 'No',
    deleteConfirmTitle: 'Confirm Delete',
    deleteConfirmDesc: 'Are you sure you want to delete this {{moduleName}}?',
    createSuccess: 'Created successfully',
    updateSuccess: 'Updated successfully',
    deleteSuccess: 'Deleted successfully',
    createFailed: 'Creation failed',
    updateFailed: 'Update failed',
    deleteFailed: 'Delete failed',
    fetchFailed: 'Failed to fetch data',
  }
}

/**
 * 生成中文按钮消息（简化版，仅 create/update）
 * @param moduleDesc 模块中文名
 */
export function buildButtonMessagesZhCN(moduleDesc: string): Record<string, string> {
  return {
    create: `创建${moduleDesc}`,
    update: `更新${moduleDesc}`,
  }
}

/**
 * 生成英文按钮消息（简化版，仅 create/update）
 * @param moduleName 模块英文名
 */
export function buildButtonMessagesEnUS(moduleName: string): Record<string, string> {
  return {
    create: `Create ${moduleName}`,
    update: `Update ${moduleName}`,
  }
}

// ==============================
// 服务描述工具函数
// ==============================

/**
 * 从服务描述中提取模块中文名
 * 例如 "角色管理服务" -> "角色"
 */
export function extractModuleName(service: ParsedService): string {
  return service.description
    .replace(/管理$/, '')
    .replace(/服务$/, '')
    .replace(/查询$/, '')
    .trim()
}

/**
 * 查找服务中的状态枚举字段
 */
export function findStatusField(service: ParsedService): ParsedField | undefined {
  return service.fields.find(f => f.isEnum && f.name.toLowerCase().includes('status'))
}

/**
 * 生成状态枚举的中文映射
 * 例如 { ON: '启用', OFF: '禁用' }
 */
export function buildStatusMapZhCN(enumValues: string[]): Record<string, string> {
  const map: Record<string, string> = {}
  for (const v of enumValues) {
    map[v] = v === 'ON' ? '启用' : v === 'OFF' ? '禁用' : v
  }
  return map
}

/**
 * 生成状态枚举的英文映射
 * 例如 { ON: 'Active', OFF: 'Inactive' }
 */
export function buildStatusMapEnUS(enumValues: string[]): Record<string, string> {
  const map: Record<string, string> = {}
  for (const v of enumValues) {
    map[v] = v === 'ON' ? 'Active' : v === 'OFF' ? 'Inactive' : v
  }
  return map
}

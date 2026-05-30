// Vue3 Element Plus - 国际化文件代码生成器
// 生成 locales/zh-CN/pages/*.json 和 locales/en-US/pages/*.json
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { serviceToFileName, toPascalCase, toCamelCase } from '../../utils/openapi-parser'

export interface LocalesTemplateOptions {
  service: ParsedService
}

// 中文描述的简单翻译映射
const zhToEnMap: Record<string, string> = {
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

  // 策略相关
  '策略名称': 'Policy Name',
  '最大登录尝试': 'Max Login Attempts',
  '锁定时长(分钟)': 'Lock Duration (min)',
  '密码最小长度': 'Min Password Length',
  '密码过期天数': 'Password Expire Days',

  // 消息相关
  '消息分类': 'Category',
  '发送者': 'Sender',
  '接收者': 'Receiver',
  '已读': 'Is Read',
  '发送时间': 'Sent At',
}

/**
 * 将中文字段描述翻译为英文
 */
function translateToEn(zhText: string, fieldName: string): string {
  // 先精确匹配
  if (zhToEnMap[zhText]) return zhToEnMap[zhText]

  // 基于 fieldName 的通用推断
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

  // 默认返回 PascalCase
  return toPascalCase(fieldName)
}

/**
 * 生成中文 locale JSON (pages/*.json)
 */
export function generateLocaleZhCN(options: LocalesTemplateOptions): string {
  const { service } = options
  const modelPascal = toPascalCase(service.modelName)

  const entries: Record<string, any> = {
    moduleName: service.description.replace(/管理$/, '').replace(/服务$/, '').trim(),
  }

  for (const field of service.fields) {
    if (field.name === 'id') continue
    entries[field.name] = field.description || field.name
  }

  // 添加按钮
  entries['button'] = {
    create: '创建',
    update: '更新',
  }

  return JSON.stringify(entries, null, 2) + '\n'
}

/**
 * 生成英文 locale JSON (pages/*.json)
 */
export function generateLocaleEnUS(options: LocalesTemplateOptions): string {
  const { service } = options
  const modelPascal = toPascalCase(service.modelName)

  const entries: Record<string, any> = {
    moduleName: service.modelName,
  }

  for (const field of service.fields) {
    if (field.name === 'id') continue
    entries[field.name] = translateToEn(field.description, field.name)
  }

  // 添加按钮
  entries['button'] = {
    create: 'Create',
    update: 'Update',
  }

  return JSON.stringify(entries, null, 2) + '\n'
}

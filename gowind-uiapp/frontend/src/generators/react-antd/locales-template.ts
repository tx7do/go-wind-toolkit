/**
 * React Ant Design Pro - 国际化文件代码生成器
 * 生成 locales/zh-CN/_modules/*.json 和 locales/en-US/_modules/*.json
 * 命名空间格式与 React 样本项目一致
 */
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { serviceToFileName, toPascalCase, toCamelCase } from '../../utils/openapi-parser'

export interface ReactLocalesTemplateOptions {
  service: ParsedService
}

// 中文描述的简单翻译映射（复用 Vue 版本）
const zhToEnMap: Record<string, string> = {
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
  '用户名': 'Username',
  '昵称': 'Nickname',
  '邮箱': 'Email',
  '手机号': 'Mobile',
  '密码': 'Password',
  '角色名称': 'Role Name',
  '角色编码': 'Role Code',
  '权限配置': 'Permissions',
  '所属租户': 'Tenant',
  '类型名称': 'Type Name',
  '类型编码': 'Type Code',
  '标签': 'Label',
  '值': 'Value',
  '文件名': 'File Name',
  '文件大小': 'File Size',
  '租户名称': 'Tenant Name',
  '联系人': 'Contact',
  '联系电话': 'Contact Phone',
  '域名': 'Domain',
  '菜单名称': 'Menu Name',
  '菜单路径': 'Menu Path',
  '菜单图标': 'Menu Icon',
  '是否启用': 'Is Enabled',
  '是否默认': 'Is Default',
  '是否外链': 'Is External',
  '是否缓存': 'Is KeepAlive',
  '是否可见': 'Is Visible',
  '请求方法': 'Request Method',
  '请求路径': 'Request Path',
  'IP地址': 'IP Address',
  '是否成功': 'Is Success',
  '操作者': 'Operator',
  '任务类型': 'Task Type',
  'Cron表达式': 'Cron Spec',
  '启用': 'Active',
  '禁用': 'Inactive',
  '是': 'Yes',
  '否': 'No',
  '语言代码': 'Language Code',
  '语言名称': 'Language Name',
  '本地名称': 'Native Name',
  '策略名称': 'Policy Name',
  '最大登录尝试': 'Max Login Attempts',
  '锁定时长(分钟)': 'Lock Duration (min)',
  '密码最小长度': 'Min Password Length',
  '密码过期天数': 'Password Expire Days',
  '消息分类': 'Category',
  '发送者': 'Sender',
  '接收者': 'Receiver',
  '已读': 'Is Read',
  '发送时间': 'Sent At',
  '组织名称': 'Organization Name',
  '职位名称': 'Position Name',
  '父级组织': 'Parent Organization',
  '管理员': 'Admin',
  '负责人': 'Leader',
  '受保护角色': 'Is Protected',
  '权限点': 'Permission Point',
  '真实姓名': 'Real Name',
  '头像': 'Avatar',
  '性别': 'Gender',
  '住址': 'Address',
  '个人描述': 'Personal Description',
  '最后登录时间': 'Last Login At',
  '最后登录IP': 'Last Login IP',
  '锁定截止时间': 'Locked Until',
  '座机号': 'Telephone',
  '数值': 'Numeric Value',
  '多语言配置': 'I18n Configuration',
  '文件类型': 'File Type',
  '存储路径': 'Storage Path',
  'MIME类型': 'MIME Type',
  '租户ID': 'Tenant ID',
  '联系邮箱': 'Contact Email',
  '有效期': 'Expired At',
  '成员数量': 'Member Count',
  '订阅时间': 'Subscription At',
  '订阅套餐': 'Subscription Plan',
  '审核状态': 'Audit Status',
  '组件路径': 'Component Path',
  '重定向': 'Redirect',
  '父级菜单': 'Parent Menu',
  '响应状态码': 'Response Status Code',
  '响应内容': 'Response Body',
  'User-Agent': 'User-Agent',
  '耗时': 'Duration',
  '任务数据': 'Task Payload',
  '序号': '#',
}

function translateToEn(zhText: string, fieldName: string): string {
  if (zhToEnMap[zhText]) return zhToEnMap[zhText]

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

/**
 * 生成中文 locale JSON
 */
export function generateReactLocaleZhCN(options: ReactLocalesTemplateOptions): string {
  const { service } = options
  const fileName = serviceToFileName(service.tagName)
  const modelDesc = service.description.replace(/管理$/, '').replace(/服务$/, '').trim()

  const entries: Record<string, any> = {
    pageTitle: `${modelDesc}管理`,
    moduleName: modelDesc,
    serial: '序号',
  }

  for (const field of service.fields) {
    if (field.name === 'id') continue
    entries[field.name] = field.description || field.name
    entries[`${field.name}Placeholder`] = `请输入${field.description || field.name}`
    if (!field.isBoolean && !['sortOrder', 'description', 'remark'].includes(field.name)) {
      entries[`required${toPascalCase(field.name)}`] = `请输入${field.description || field.name}`
    }
  }

  // 通用按钮和消息
  entries['action'] = '操作'
  entries['create'] = `新建${modelDesc}`
  entries['edit'] = `编辑${modelDesc}`
  entries['yes'] = '是'
  entries['no'] = '否'
  entries['deleteConfirmTitle'] = '确认删除'
  entries['deleteConfirmDesc'] = `确定要删除该{{moduleName}}吗？`
  entries['createSuccess'] = '创建成功'
  entries['updateSuccess'] = '更新成功'
  entries['deleteSuccess'] = '删除成功'
  entries['createFailed'] = '创建失败'
  entries['updateFailed'] = '更新失败'
  entries['deleteFailed'] = '删除失败'
  entries['fetchFailed'] = '获取数据失败'

  // 状态映射
  const statusField = service.fields.find(f => f.isEnum && f.name.toLowerCase().includes('status'))
  if (statusField?.enumValues) {
    const statusMap: Record<string, string> = {}
    for (const v of statusField.enumValues) {
      statusMap[v] = v === 'ON' ? '启用' : v === 'OFF' ? '禁用' : v
    }
    entries['statusMap'] = statusMap
  }

  return JSON.stringify(entries, null, 2) + '\n'
}

/**
 * 生成英文 locale JSON
 */
export function generateReactLocaleEnUS(options: ReactLocalesTemplateOptions): string {
  const { service } = options

  const entries: Record<string, any> = {
    pageTitle: `${service.modelName} Management`,
    moduleName: service.modelName,
    serial: '#',
  }

  for (const field of service.fields) {
    if (field.name === 'id') continue
    const enName = translateToEn(field.description, field.name)
    entries[field.name] = enName
    entries[`${field.name}Placeholder`] = `Enter ${enName.toLowerCase()}`
    if (!field.isBoolean && !['sortOrder', 'description', 'remark'].includes(field.name)) {
      entries[`required${toPascalCase(field.name)}`] = `Please enter ${enName.toLowerCase()}`
    }
  }

  entries['action'] = 'Actions'
  entries['create'] = `New ${service.modelName}`
  entries['edit'] = `Edit ${service.modelName}`
  entries['yes'] = 'Yes'
  entries['no'] = 'No'
  entries['deleteConfirmTitle'] = 'Confirm Delete'
  entries['deleteConfirmDesc'] = 'Are you sure you want to delete this {{moduleName}}?'
  entries['createSuccess'] = 'Created successfully'
  entries['updateSuccess'] = 'Updated successfully'
  entries['deleteSuccess'] = 'Deleted successfully'
  entries['createFailed'] = 'Creation failed'
  entries['updateFailed'] = 'Update failed'
  entries['deleteFailed'] = 'Delete failed'
  entries['fetchFailed'] = 'Failed to fetch data'

  const statusField = service.fields.find(f => f.isEnum && f.name.toLowerCase().includes('status'))
  if (statusField?.enumValues) {
    const statusMap: Record<string, string> = {}
    for (const v of statusField.enumValues) {
      statusMap[v] = v === 'ON' ? 'Active' : v === 'OFF' ? 'Inactive' : v
    }
    entries['statusMap'] = statusMap
  }

  return JSON.stringify(entries, null, 2) + '\n'
}

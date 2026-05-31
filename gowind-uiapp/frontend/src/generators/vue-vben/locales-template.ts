/**
 * Vue Vben - 国际化代码生成器
 * 生成 page.json 片段（zh-CN / en-US）
 * Vben 使用合并式 page.json 和 menu.json，非命名空间文件
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toCamelCase } from '../../utils/openapi-parser'

export interface VbenLocaleTemplateOptions {
  service: ParsedService
}

// ==============================
// 中文字段名到英文的翻译映射
// ==============================
const zhToEnMap: Record<string, string> = {
  '角色名称': 'Role Name',
  '角色值': 'Role Code',
  '角色描述': 'Role Description',
  '权限名称': 'Permission Name',
  '权限编码': 'Permission Code',
  '组织名称': 'Organization Name',
  '唯一编码': 'Unique Code',
  '用户名': 'Username',
  '昵称': 'Nickname',
  '真实姓名': 'Real Name',
  '电子邮箱': 'Email',
  '手机号码': 'Mobile',
  '密码': 'Password',
  '状态': 'Status',
  '描述': 'Description',
  '备注': 'Remark',
  '排序': 'Sort Order',
  '标题': 'Title',
  '内容': 'Content',
  '类型': 'Type',
  '图标': 'Icon',
  '路径': 'Path',
  '方法': 'Method',
  '标签': 'Label',
  '值': 'Value',
  '名称': 'Name',
  '编码': 'Code',
  '语言名称': 'Language Name',
  '语言代码': 'Language Code',
  '本地名称': 'Native Name',
  '是否启用': 'Is Enabled',
  '是否默认': 'Is Default',
  '租户名称': 'Tenant Name',
  '租户编码': 'Tenant Code',
  '创建时间': 'Created At',
  '更新时间': 'Updated At',
  '排序值': 'Sort Order',
  '分类名称': 'Category Name',
  '分类编码': 'Category Code',
  '消息主题': 'Subject',
  '消息内容': 'Content',
  '消息类型': 'Type',
  '消息状态': 'Status',
}

/**
 * 根据字段描述和字段名推断英文翻译
 */
function translateToEn(fieldDesc: string, fieldName: string): string {
  // 先从映射表找
  if (zhToEnMap[fieldDesc]) return zhToEnMap[fieldDesc]

  // 根据字段名推断
  const nameToEn: Record<string, string> = {
    name: 'Name',
    code: 'Code',
    title: 'Title',
    description: 'Description',
    remark: 'Remark',
    status: 'Status',
    sortOrder: 'Sort Order',
    isEnabled: 'Is Enabled',
    isDefault: 'Is Default',
    createdAt: 'Created At',
    updatedAt: 'Updated At',
  }
  if (nameToEn[fieldName]) return nameToEn[fieldName]

  // 生成默认的 PascalCase
  return fieldName.replace(/([A-Z])/g, ' $1').replace(/^./, s => s.toUpperCase()).trim()
}

/**
 * 生成中文 page.json 片段
 */
export function generateVbenLocalePageZhCN(options: VbenLocaleTemplateOptions): string {
  const { service } = options
  const modelCamel = toCamelCase(service.modelName)
  const modelDesc = service.description || service.modelName

  // 提取模块中文名
  const moduleName = modelDesc.replace(/管理.*/, '').replace(/服务.*/, '').replace(/查询.*/, '').trim()

  const lines: string[] = []
  lines.push(`  "${modelCamel}": {`)
  lines.push(`    "moduleName": "${moduleName}",`)

  for (const field of service.fields) {
    if (['id', 'createdAt', 'updatedAt'].includes(field.name)) continue
    lines.push(`    "${field.name}": "${field.description || field.name}",`)
  }

  lines.push(`    "button": {`)
  lines.push(`      "create": "创建${moduleName}",`)
  lines.push(`      "update": "更新${moduleName}"`)
  lines.push(`    }`)
  lines.push(`  }`)

  return lines.join('\n')
}

/**
 * 生成英文 page.json 片段
 */
export function generateVbenLocalePageEnUS(options: VbenLocaleTemplateOptions): string {
  const { service } = options
  const modelCamel = toCamelCase(service.modelName)
  const modelDesc = service.description || service.modelName

  // 提取模块英文名
  const moduleName = translateToEn(modelDesc.replace(/管理.*/, '').replace(/服务.*/, '').replace(/查询.*/, '').trim(), modelCamel)

  const lines: string[] = []
  lines.push(`  "${modelCamel}": {`)
  lines.push(`    "moduleName": "${moduleName}",`)

  for (const field of service.fields) {
    if (['id', 'createdAt', 'updatedAt'].includes(field.name)) continue
    const enName = translateToEn(field.description || '', field.name)
    lines.push(`    "${field.name}": "${enName}",`)
  }

  lines.push(`    "button": {`)
  lines.push(`      "create": "Create ${moduleName}",`)
  lines.push(`      "update": "Update ${moduleName}"`)
  lines.push(`    }`)
  lines.push(`  }`)

  return lines.join('\n')
}

/**
 * 生成中文 menu.json 片段（路由模块级别）
 */
export function generateVbenLocaleMenuZhCN(moduleKey: string, moduleDisplayName: string, services: ParsedService[]): string {
  const lines: string[] = []
  lines.push(`  "${moduleKey}": {`)
  lines.push(`    "moduleName": "${moduleDisplayName}",`)

  for (const service of services) {
    const modelCamel = toCamelCase(service.modelName)
    const modelDesc = service.description || service.modelName
    const modelName = modelDesc.replace(/管理.*/, '').replace(/服务.*/, '').replace(/查询.*/, '').trim()
    lines.push(`    "${modelCamel}": "${modelName}管理",`)
  }

  lines.push(`  }`)
  return lines.join('\n')
}

/**
 * 生成英文 menu.json 片段
 */
export function generateVbenLocaleMenuEnUS(moduleKey: string, moduleDisplayName: string, services: ParsedService[]): string {
  const lines: string[] = []
  lines.push(`  "${moduleKey}": {`)
  lines.push(`    "moduleName": "${moduleDisplayName}",`)

  for (const service of services) {
    const modelCamel = toCamelCase(service.modelName)
    const enName = translateToEn(service.description || '', modelCamel)
    lines.push(`    "${modelCamel}": "${enName}",`)
  }

  lines.push(`  }`)
  return lines.join('\n')
}

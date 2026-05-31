/**
 * Vue Vben - 国际化代码生成器
 * 生成 page.json 片段（zh-CN / en-US）
 * Vben 使用合并式 page.json 和 menu.json，非命名空间文件
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toCamelCase } from '../../utils/openapi-parser'
import {
  translateToEn,
  extractModuleName,
  buildFieldEntriesZhCN,
  buildFieldEntriesEnUS,
  buildButtonMessagesZhCN,
  buildButtonMessagesEnUS,
} from '../../utils/i18n-translator'

export interface VbenLocaleTemplateOptions {
  service: ParsedService
}

/** Vben 跳过的字段：id + 系统时间戳 */
const vbenFieldOptions = { skipFields: ['id', 'createdAt', 'updatedAt'] }

/**
 * 生成中文 page.json 片段
 */
export function generateVbenLocalePageZhCN(options: VbenLocaleTemplateOptions): string {
  const { service } = options
  const modelCamel = toCamelCase(service.modelName)
  const moduleName = extractModuleName(service)

  const fieldEntries = buildFieldEntriesZhCN(service.fields, vbenFieldOptions)
  const buttons = buildButtonMessagesZhCN(moduleName)

  const lines: string[] = []
  lines.push(`  "${modelCamel}": {`)
  lines.push(`    "moduleName": "${moduleName}",`)
  for (const [key, val] of Object.entries(fieldEntries)) {
    lines.push(`    "${key}": "${val}",`)
  }
  lines.push(`    "button": {`)
  lines.push(`      "create": "${buttons.create}",`)
  lines.push(`      "update": "${buttons.update}"`)
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
  const moduleName = translateToEn(extractModuleName(service), modelCamel)

  const fieldEntries = buildFieldEntriesEnUS(service.fields, vbenFieldOptions)
  const buttons = buildButtonMessagesEnUS(moduleName)

  const lines: string[] = []
  lines.push(`  "${modelCamel}": {`)
  lines.push(`    "moduleName": "${moduleName}",`)
  for (const [key, val] of Object.entries(fieldEntries)) {
    lines.push(`    "${key}": "${val}",`)
  }
  lines.push(`    "button": {`)
  lines.push(`      "create": "${buttons.create}",`)
  lines.push(`      "update": "${buttons.update}"`)
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
    const modelName = extractModuleName(service)
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

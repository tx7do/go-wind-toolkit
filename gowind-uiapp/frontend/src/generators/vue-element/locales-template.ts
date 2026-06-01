// Vue3 Element Plus - 国际化文件代码生成器
// 生成 locales/zh-CN/pages/*.json 和 locales/en-US/pages/*.json
import type { ParsedService } from '../../utils/openapi-parser'
import {
  extractModuleName,
  buildFieldEntriesZhCN,
  buildFieldEntriesEnUS,
  buildButtonMessagesZhCN,
  buildButtonMessagesEnUS,
} from '../../utils/i18n-translator'

export interface LocalesTemplateOptions {
  service: ParsedService
}

/**
 * 生成中文 locale JSON (pages/*.json)
 */
export function generateLocaleZhCN(options: LocalesTemplateOptions): string {
  const { service } = options

  const entries: Record<string, any> = {
    moduleName: extractModuleName(service),
    ...buildFieldEntriesZhCN(service.fields),
  }

  entries['button'] = buildButtonMessagesZhCN(extractModuleName(service))

  return JSON.stringify(entries, null, 2) + '\n'
}

/**
 * 生成英文 locale JSON (pages/*.json)
 */
export function generateLocaleEnUS(options: LocalesTemplateOptions): string {
  const { service } = options

  const entries: Record<string, any> = {
    moduleName: service.modelName,
    ...buildFieldEntriesEnUS(service.fields),
  }

  entries['button'] = buildButtonMessagesEnUS(service.modelName)

  return JSON.stringify(entries, null, 2) + '\n'
}

/**
 * React Ant Design Pro - 国际化文件代码生成器
 * 生成 locales/zh-CN/_modules/*.json 和 locales/en-US/_modules/*.json
 * 命名空间格式与 React 样本项目一致
 */
import type { ParsedService } from '../../utils/openapi-parser'
import {
  extractModuleName,
  findStatusField,
  buildStatusMapZhCN,
  buildStatusMapEnUS,
  buildFieldEntriesZhCN,
  buildFieldEntriesEnUS,
  buildActionMessagesZhCN,
  buildActionMessagesEnUS,
} from '../../utils/i18n-translator'

export interface ReactLocalesTemplateOptions {
  service: ParsedService
}

/** react-antd 字段构建选项：带 placeholder 和 required */
const fieldOptions = { withPlaceholder: true, withRequired: true }

/**
 * 生成中文 locale JSON
 */
export function generateReactLocaleZhCN(options: ReactLocalesTemplateOptions): string {
  const { service } = options
  const modelDesc = extractModuleName(service)

  const entries: Record<string, any> = {
    pageTitle: `${modelDesc}管理`,
    moduleName: modelDesc,
    serial: '序号',
  }

  Object.assign(entries, buildFieldEntriesZhCN(service.fields, fieldOptions))
  Object.assign(entries, buildActionMessagesZhCN(modelDesc))

  const statusField = findStatusField(service)
  if (statusField?.enumValues) {
    entries['statusMap'] = buildStatusMapZhCN(statusField.enumValues)
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

  Object.assign(entries, buildFieldEntriesEnUS(service.fields, fieldOptions))
  Object.assign(entries, buildActionMessagesEnUS(service.modelName))

  const statusField = findStatusField(service)
  if (statusField?.enumValues) {
    entries['statusMap'] = buildStatusMapEnUS(statusField.enumValues)
  }

  return JSON.stringify(entries, null, 2) + '\n'
}

/**
 * Vue3 Element Plus 代码生成器
 * 统一入口，根据 OpenAPI 规范生成完整的前端代码
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { serviceToFileName, toKebabCase } from '../../utils/openapi-parser'
import { generateServiceCode } from './service-template'
import { generateComposableCode } from './composable-template'
import { generatePageCode, generateDrawerCode } from './page-template'
import { generateRouterCode, generateRoutesI18n, type RouterTemplateOptions } from './router-template'
import { generateLocaleZhCN, generateLocaleEnUS } from './locales-template'

export type GenerateFileType = 'service' | 'composable' | 'page' | 'drawer' | 'router' | 'locale'

export interface VueElementGeneratorOptions {
  /** 要生成的服务列表 */
  services: ParsedService[]
  /** 服务名（如 admin） */
  serviceName?: string
  /** 模块路径映射（serviceName -> modulePath，如 role -> permission/role） */
  modulePathMap?: Record<string, string>
  /** 要生成的文件类型 */
  generateTypes?: GenerateFileType[]
  /** 路由模块分组配置 */
  routerModules?: RouterModuleConfig[]
}

/** 路由模块分组配置 */
export interface RouterModuleConfig {
  /** 模块标识 (如 permission, system) */
  moduleKey: string
  /** 模块中文名 */
  moduleDisplayName: string
  /** 模块图标 */
  moduleIcon?: string
  /** 模块排序 */
  moduleOrder?: number
  /** 权限标识列表 */
  authority?: string[]
  /** 该模块包含的服务 tag 名列表 */
  serviceTags: string[]
}

export interface GeneratedFile {
  /** 文件相对路径 */
  path: string
  /** 文件内容 */
  content: string
  /** 文件描述 */
  description: string
  /** 所属服务 */
  serviceName: string
  /** 文件类型 */
  type: GenerateFileType
}

/**
 * 生成所有文件
 */
export function generateAll(options: VueElementGeneratorOptions): GeneratedFile[] {
  const {
    services,
    serviceName = 'admin',
    modulePathMap = {},
    generateTypes = ['service', 'composable', 'page', 'drawer', 'router', 'locale'],
    routerModules,
  } = options

  const files: GeneratedFile[] = []

  for (const service of services) {
    const fileName = serviceToFileName(service.tagName)
    const modulePath = modulePathMap[fileName] || fileName
    const commonOpts = { service, serviceName }

    if (generateTypes.includes('service')) {
      files.push({
        path: `api/service/${fileName}.ts`,
        content: generateServiceCode(commonOpts),
        description: `${service.modelName} API Service 层`,
        serviceName: service.tagName,
        type: 'service',
      })
    }

    if (generateTypes.includes('composable')) {
      files.push({
        path: `api/composables/${fileName}.ts`,
        content: generateComposableCode(commonOpts),
        description: `${service.modelName} Vue Query Composable`,
        serviceName: service.tagName,
        type: 'composable',
      })
    }

    if (generateTypes.includes('page')) {
      files.push({
        path: `pages/${modulePath}/index.vue`,
        content: generatePageCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 列表页面`,
        serviceName: service.tagName,
        type: 'page',
      })
    }

    if (generateTypes.includes('drawer')) {
      files.push({
        path: `pages/${modulePath}/${fileName}-drawer.vue`,
        content: generateDrawerCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 编辑抽屉`,
        serviceName: service.tagName,
        type: 'drawer',
      })
    }

    // 生成 locale (i18n) 文件
    if (generateTypes.includes('locale')) {
      const localeOpts = { service }
      files.push({
        path: `locales/zh-CN/pages/${fileName}.json`,
        content: generateLocaleZhCN(localeOpts),
        description: `${service.modelName} 中文国际化`,
        serviceName: service.tagName,
        type: 'locale',
      })
      files.push({
        path: `locales/en-US/pages/${fileName}.json`,
        content: generateLocaleEnUS(localeOpts),
        description: `${service.modelName} 英文国际化`,
        serviceName: service.tagName,
        type: 'locale',
      })
    }
  }

  // 生成路由文件（按模块分组）
  if (generateTypes.includes('router') && options.routerModules && options.routerModules.length > 0) {
    for (const moduleConfig of options.routerModules) {
      const moduleServices = services.filter(s => moduleConfig.serviceTags.includes(s.tagName))
      if (moduleServices.length === 0) continue

      const routerOpts: RouterTemplateOptions = {
        services: moduleServices,
        moduleName: moduleConfig.moduleDisplayName,
        moduleKey: moduleConfig.moduleKey,
        moduleDisplayName: moduleConfig.moduleDisplayName,
        moduleIcon: moduleConfig.moduleIcon,
        moduleOrder: moduleConfig.moduleOrder,
        authority: moduleConfig.authority,
      }

      files.push({
        path: `router/routes/modules/app/${moduleConfig.moduleKey}.ts`,
        content: generateRouterCode(routerOpts),
        description: `${moduleConfig.moduleDisplayName} 路由配置`,
        serviceName: moduleConfig.moduleKey,
        type: 'router',
      })
    }
  }

  return files
}

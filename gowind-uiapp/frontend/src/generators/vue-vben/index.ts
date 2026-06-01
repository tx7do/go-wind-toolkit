/**
 * Vue Vben 代码生成器
 * 统一入口，根据 OpenAPI 规范生成完整的前端代码
 * 使用 VxeGrid + useVbenDrawer + useVbenForm + @tanstack/vue-query
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { serviceToFileName, toKebabCase } from '../../utils/openapi-parser'
import { generateVbenServiceCode } from './service-template'
import { generateVbenComposableCode } from './composable-template'
import { generateVbenPageCode, generateVbenDrawerCode } from './page-template'
import { generateVbenRouterCode, type VbenRouterTemplateOptions } from './router-template'
import {
  generateVbenLocalePageZhCN,
  generateVbenLocalePageEnUS,
  generateVbenLocaleMenuZhCN,
  generateVbenLocaleMenuEnUS,
} from './locales-template'

export type VbenGenerateFileType = 'service' | 'composable' | 'page' | 'drawer' | 'router' | 'locale'

export interface VbenVbenGeneratorOptions {
  /** 要生成的服务列表 */
  services: ParsedService[]
  /** 服务名（如 admin） */
  serviceName?: string
  /** 模块路径映射（fileName -> modulePath，如 role -> permission/role） */
  modulePathMap?: Record<string, string>
  /** 要生成的文件类型 */
  generateTypes?: VbenGenerateFileType[]
  /** 路由模块分组配置 */
  routerModules?: VbenRouterModuleConfig[]
}

/** 路由模块分组配置 */
export interface VbenRouterModuleConfig {
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

export interface VbenGeneratedFile {
  /** 文件相对路径 */
  path: string
  /** 文件内容 */
  content: string
  /** 文件描述 */
  description: string
  /** 所属服务 */
  serviceName: string
  /** 文件类型 */
  type: VbenGenerateFileType
}

/**
 * 生成所有 Vben 文件
 */
export function generateAll(options: VbenVbenGeneratorOptions): VbenGeneratedFile[] {
  const {
    services,
    serviceName = 'admin',
    modulePathMap = {},
    generateTypes = ['service', 'composable', 'page', 'drawer', 'router', 'locale'],
    routerModules,
  } = options

  const files: VbenGeneratedFile[] = []

  for (const service of services) {
    const fileName = serviceToFileName(service.tagName)
    const modulePath = modulePathMap[fileName] || fileName
    const commonOpts = { service, serviceName }

    // API Service 层
    if (generateTypes.includes('service')) {
      files.push({
        path: `api/service/${fileName}.ts`,
        content: generateVbenServiceCode(commonOpts),
        description: `${service.modelName} API Service 层`,
        serviceName: service.tagName,
        type: 'service',
      })
    }

    // Vue Query Composable
    if (generateTypes.includes('composable')) {
      files.push({
        path: `api/composables/${fileName}.ts`,
        content: generateVbenComposableCode(commonOpts),
        description: `${service.modelName} Vue Query Composable`,
        serviceName: service.tagName,
        type: 'composable',
      })
    }

    // VxeGrid 列表页
    if (generateTypes.includes('page')) {
      files.push({
        path: `views/app/${modulePath}/index.vue`,
        content: generateVbenPageCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 列表页面`,
        serviceName: service.tagName,
        type: 'page',
      })
    }

    // 编辑抽屉
    if (generateTypes.includes('drawer')) {
      files.push({
        path: `views/app/${modulePath}/${fileName}-drawer.vue`,
        content: generateVbenDrawerCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 编辑抽屉`,
        serviceName: service.tagName,
        type: 'drawer',
      })
    }

    // 国际化文件（page.json 片段）
    if (generateTypes.includes('locale')) {
      const localeOpts = { service }
      files.push({
        path: `locales/langs/zh-CN/page.${fileName}.json`,
        content: generateVbenLocalePageZhCN(localeOpts),
        description: `${service.modelName} 中文国际化`,
        serviceName: service.tagName,
        type: 'locale',
      })
      files.push({
        path: `locales/langs/en-US/page.${fileName}.json`,
        content: generateVbenLocalePageEnUS(localeOpts),
        description: `${service.modelName} 英文国际化`,
        serviceName: service.tagName,
        type: 'locale',
      })
    }
  }

  // 路由文件（按模块分组）
  if (generateTypes.includes('router') && routerModules && routerModules.length > 0) {
    for (const moduleConfig of routerModules) {
      const moduleServices = services.filter(s => moduleConfig.serviceTags.includes(s.tagName))
      if (moduleServices.length === 0) continue

      const routerOpts: VbenRouterTemplateOptions = {
        services: moduleServices,
        moduleKey: moduleConfig.moduleKey,
        moduleDisplayName: moduleConfig.moduleDisplayName,
        moduleIcon: moduleConfig.moduleIcon,
        moduleOrder: moduleConfig.moduleOrder,
        authority: moduleConfig.authority,
        modulePathMap,
      }

      files.push({
        path: `router/routes/modules/app/${moduleConfig.moduleKey}.ts`,
        content: generateVbenRouterCode(routerOpts),
        description: `${moduleConfig.moduleDisplayName} 路由配置`,
        serviceName: moduleConfig.moduleKey,
        type: 'router',
      })

      // 路由对应的 menu.json 国际化片段
      if (generateTypes.includes('locale')) {
        files.push({
          path: `locales/langs/zh-CN/menu.${moduleConfig.moduleKey}.json`,
          content: generateVbenLocaleMenuZhCN(
            moduleConfig.moduleKey,
            moduleConfig.moduleDisplayName,
            moduleServices,
          ),
          description: `${moduleConfig.moduleDisplayName} 菜单中文国际化`,
          serviceName: moduleConfig.moduleKey,
          type: 'locale',
        })
        files.push({
          path: `locales/langs/en-US/menu.${moduleConfig.moduleKey}.json`,
          content: generateVbenLocaleMenuEnUS(
            moduleConfig.moduleKey,
            moduleConfig.moduleDisplayName,
            moduleServices,
          ),
          description: `${moduleConfig.moduleDisplayName} 菜单英文国际化`,
          serviceName: moduleConfig.moduleKey,
          type: 'locale',
        })
      }
    }
  }

  return files
}

/**
 * React Ant Design Pro 代码生成器
 * 统一入口，根据 OpenAPI 规范生成完整的前端代码
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { serviceToFileName, toPascalCase } from '../../utils/openapi-parser'
import { generateServiceCode } from './service-template'
import { generateHooksCode } from './hooks-template'
import { generatePageCode, generateDrawerCode, generateConstantsCode } from './page-template'
import { generateReactRouterCode, type ReactRouterTemplateOptions } from './router-template'
import { generateReactLocaleZhCN, generateReactLocaleEnUS } from './locales-template'

export type ReactGenerateFileType = 'service' | 'hooks' | 'page' | 'drawer' | 'router' | 'locale'

export interface ReactAntdGeneratorOptions {
  /** 要生成的服务列表 */
  services: ParsedService[]
  /** 服务名（如 admin） */
  serviceName?: string
  /** 模块路径映射（serviceName -> modulePath，如 role -> permission/role） */
  modulePathMap?: Record<string, string>
  /** 要生成的文件类型 */
  generateTypes?: ReactGenerateFileType[]
  /** 路由模块分组配置 */
  routerModules?: ReactRouterModuleConfig[]
}

/** 路由模块分组配置 */
export interface ReactRouterModuleConfig {
  /** 模块标识 */
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

export interface ReactGeneratedFile {
  /** 文件相对路径 */
  path: string
  /** 文件内容 */
  content: string
  /** 文件描述 */
  description: string
  /** 所属服务 */
  serviceName: string
  /** 文件类型 */
  type: ReactGenerateFileType
}

/**
 * 生成所有 React 文件
 */
export function generateAll(options: ReactAntdGeneratorOptions): ReactGeneratedFile[] {
  const {
    services,
    serviceName = 'admin',
    modulePathMap = {},
    generateTypes = ['service', 'hooks', 'page', 'drawer', 'router', 'locale'],
    routerModules,
  } = options

  const files: ReactGeneratedFile[] = []

  for (const service of services) {
    const fileName = serviceToFileName(service.tagName).replace(/-/g, '')
    const modulePath = modulePathMap[fileName] || fileName
    const commonOpts = { service, serviceName }

    // API Service 层
    if (generateTypes.includes('service')) {
      files.push({
        path: `api/service/${fileName}.ts`,
        content: generateServiceCode(commonOpts),
        description: `${service.modelName} API Service 层`,
        serviceName: service.tagName,
        type: 'service',
      })
    }

    // React Query Hooks
    if (generateTypes.includes('hooks')) {
      files.push({
        path: `api/hooks/${fileName}.ts`,
        content: generateHooksCode(commonOpts),
        description: `${service.modelName} React Query Hooks`,
        serviceName: service.tagName,
        type: 'hooks',
      })
    }

    // ProTable 列表页
    if (generateTypes.includes('page')) {
      files.push({
        path: `pages/app/${modulePath}/index.tsx`,
        content: generatePageCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 列表页面`,
        serviceName: service.tagName,
        type: 'page',
      })
    }

    // DrawerForm 编辑抽屉
    if (generateTypes.includes('drawer')) {
      files.push({
        path: `pages/app/${modulePath}/components/${toPascalCase(service.modelName)}Drawer.tsx`,
        content: generateDrawerCode({ ...commonOpts, modulePath }),
        description: `${service.modelName} 编辑抽屉`,
        serviceName: service.tagName,
        type: 'drawer',
      })

      // constants.ts（如果包含 status 枚举）
      const constantsCode = generateConstantsCode(service)
      if (constantsCode) {
        files.push({
          path: `pages/app/${modulePath}/constants.ts`,
          content: constantsCode,
          description: `${service.modelName} 常量定义`,
          serviceName: service.tagName,
          type: 'drawer',
        })
      }
    }

    // 国际化文件
    if (generateTypes.includes('locale')) {
      files.push({
        path: `locales/zh-CN/_modules/${fileName}.json`,
        content: generateReactLocaleZhCN({ service }),
        description: `${service.modelName} 中文国际化`,
        serviceName: service.tagName,
        type: 'locale',
      })
      files.push({
        path: `locales/en-US/_modules/${fileName}.json`,
        content: generateReactLocaleEnUS({ service }),
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

      const routerOpts: ReactRouterTemplateOptions = {
        services: moduleServices,
        moduleKey: moduleConfig.moduleKey,
        moduleDisplayName: moduleConfig.moduleDisplayName,
        moduleIcon: moduleConfig.moduleIcon,
        moduleOrder: moduleConfig.moduleOrder,
        authority: moduleConfig.authority,
      }

      files.push({
        path: `router/modules/${moduleConfig.moduleKey}.tsx`,
        content: generateReactRouterCode(routerOpts),
        description: `${moduleConfig.moduleDisplayName} 路由配置`,
        serviceName: moduleConfig.moduleKey,
        type: 'router',
      })
    }
  }

  return files
}


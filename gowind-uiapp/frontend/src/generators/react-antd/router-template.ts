/**
 * React Ant Design Pro - 路由代码生成器
 * 生成 router/modules/*.tsx 文件
 * 风格与 React 样本项目一致（createLazyRoute + AppRouteObject）
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toKebabCase, toPascalCase, serviceToFileName, getCrudPaths } from '../../utils/openapi-parser'

export interface ReactRouterTemplateOptions {
  /** 同一模块下的服务列表 */
  services: ParsedService[]
  /** 模块英文标识 (如 permission, system) */
  moduleKey: string
  /** 模块中文名 */
  moduleDisplayName: string
  /** 模块图标 */
  moduleIcon?: string
  /** 模块排序 */
  moduleOrder?: number
  /** 权限标识列表 */
  authority?: string[]
}

/**
 * 根据服务名推断 Lucide 图标
 */
function inferIcon(service: ParsedService): string {
  const name = service.modelName.toLowerCase()
  const iconMap: Record<string, string> = {
    'role': 'lucide:shield-user',
    'permission': 'lucide:shield-ellipsis',
    'permissiongroup': 'lucide:shield-plus',
    'menu': 'lucide:square-menu',
    'api': 'lucide:route',
    'user': 'lucide:user',
    'orgunit': 'lucide:layers',
    'position': 'lucide:briefcase',
    'dicttype': 'lucide:library-big',
    'dictentry': 'lucide:list',
    'file': 'lucide:file-search',
    'task': 'lucide:list-todo',
    'loginpolicy': 'lucide:shield-x',
    'language': 'lucide:globe',
    'tenant': 'lucide:building-2',
    'loginauditlog': 'lucide:scroll-text',
    'apiauditlog': 'lucide:file-text',
    'operationauditlog': 'lucide:file-clock',
    'dataaccessauditlog': 'lucide:database',
    'permissionauditlog': 'lucide:shield-alert',
    'internalmessage': 'lucide:message-square',
    'internalmessagecategory': 'lucide:folder',
  }
  return iconMap[name] || 'lucide:file'
}

/**
 * 生成路由文件代码（.tsx）
 */
export function generateReactRouterCode(options: ReactRouterTemplateOptions): string {
  const {
    services,
    moduleKey,
    moduleDisplayName,
    moduleIcon = 'lucide:folder',
    moduleOrder = 2000,
    authority = [],
  } = options

  const children: string[] = []
  let order = 1

  for (const service of services) {
    const crudPaths = getCrudPaths(service)
    if (!crudPaths.list) continue

    const serviceFileName = serviceToFileName(service.tagName)
    const serviceKebab = serviceFileName.replace(/-/g, '')
    const routeName = `${moduleKey}-${serviceKebab}`

    const icon = inferIcon(service)
    const routeAuthority = authority.length > 0
      ? `\n          // permission: '${authority[0]}',`
      : ''

    children.push(`      {
        name: '${routeName}',
        path: '${serviceFileName}',
        element: createLazyRoute(() => import('@/pages/app/${moduleKey}/${serviceFileName}')),
        meta: {
          title: 'routes:${moduleKey}-${serviceFileName}',
          icon: '${icon}',
          order: ${order},${routeAuthority}
        },
      },`)

    order++
  }

  if (children.length === 0) {
    return `// ${moduleDisplayName} - 没有可用的服务（需要 List 操作）`
  }

  const parentAuthority = authority.length > 0
    ? `\n      // permission: '${authority[0]}',`
    : ''

  let code = `import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * ${moduleDisplayName}路由配置
 */
export const ${moduleKey}Routes: AppRouteObject[] = [
  {
    name: '${moduleKey}',
    path: '${moduleKey}',
    meta: {
      title: 'routes:${moduleKey}',
      icon: '${moduleIcon}',
      order: ${moduleOrder},
      keepAlive: true,${parentAuthority}
    },
    children: [
${children.join(',\n')}
    ],
  },
];

export default ${moduleKey}Routes;
`
  return code
}

/**
 * 生成路由 i18n 片段（用于 routes.json）
 */
export function generateReactRoutesI18n(options: ReactRouterTemplateOptions): {
  zhCN: Record<string, string>
  enUS: Record<string, string>
} {
  const { services, moduleKey, moduleDisplayName } = options

  const zhCN: Record<string, string> = {
    [moduleKey]: moduleDisplayName,
  }
  const enUS: Record<string, string> = {
    [moduleKey]: moduleDisplayName + ' Management',
  }

  for (const service of services) {
    const crudPaths = getCrudPaths(service)
    if (!crudPaths.list) continue

    const serviceFileName = serviceToFileName(service.tagName)
    zhCN[`${moduleKey}-${serviceFileName}`] = service.description
    enUS[`${moduleKey}-${serviceFileName}`] = service.modelName + ' Management'
  }

  return { zhCN, enUS }
}

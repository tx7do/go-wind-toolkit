// Vue3 Element Plus - 路由代码生成器
// 生成 router/routes/modules/app/*.ts 文件
import type { ParsedService } from '../../utils/openapi-parser'
import { toKebabCase, toPascalCase, serviceToFileName, getCrudPaths } from '../../utils/openapi-parser'

export interface RouterTemplateOptions {
  /** 同一模块下的服务列表 */
  services: ParsedService[]
  /** 模块名 (如 permission, system, opm) */
  moduleName: string
  /** 模块英文标识 (用于路由路径和i18n key) */
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
 * 生成路由文件代码
 */
export function generateRouterCode(options: RouterTemplateOptions): string {
  const {
    services,
    moduleName,
    moduleKey,
    moduleDisplayName,
    moduleIcon = 'lucide:folder',
    moduleOrder = 2000,
    authority = [],
  } = options

  const modulePascal = toPascalCase(moduleKey)
  const moduleKebab = toKebabCase(moduleKey)

  let code = `import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const ${moduleKey}: RouteRecordRaw[] = [
  {
    path: "/${moduleKebab}",
    name: "${modulePascal}Management",
    component: Layout,
    redirect: "/${moduleKebab}/${toKebabCase(serviceToFileName(services[0]?.tagName || ''))}",
    meta: {
      order: ${moduleOrder},
      icon: "${moduleIcon}",
      title: "routes.${moduleKey}.moduleName",
      keepAlive: true,${authority.length > 0 ? `\n      authority: ${JSON.stringify(authority)},` : ''}
    },
    children: [`

  const children: string[] = []
  let order = 1

  for (const service of services) {
    const crudPaths = getCrudPaths(service)
    // 只为有 list 操作的服务生成路由
    if (!crudPaths.list) continue

    const serviceFileName = serviceToFileName(service.tagName)
    const serviceKebab = toKebabCase(serviceFileName)
    const servicePascal = toPascalCase(serviceFileName) + 'Management'

    const modulePath = service.basePath
      ? service.basePath.split('/').pop() || serviceKebab
      : serviceKebab

    const icon = inferIcon(service)
    const routeAuthority = authority.length > 0 ? `\n          authority: ${JSON.stringify(authority)},` : ''

    children.push(`
      {
        path: "${serviceKebab}",
        name: "${servicePascal}",
        meta: {
          order: ${order},
          icon: "${icon}",
          title: "routes.${moduleKey}.${serviceFileName}",${routeAuthority}
        },
        component: () => import("@/pages/app/${moduleKey}/${serviceFileName}/index.vue"),
      },`)

    order++
  }

  code += children.join(',')
  code += `
    ],
  },
];

export default ${moduleKey};
`

  return code
}

/**
 * 生成路由 i18n 片段（用于 routes.json）
 */
export function generateRoutesI18n(options: RouterTemplateOptions): {
  zhCN: Record<string, string>
  enUS: Record<string, string>
} {
  const { services, moduleKey, moduleDisplayName } = options

  const zhCN: Record<string, string> = {
    moduleName: moduleDisplayName,
  }
  const enUS: Record<string, string> = {
    moduleName: moduleDisplayName + ' Management',
  }

  for (const service of services) {
    const crudPaths = getCrudPaths(service)
    if (!crudPaths.list) continue

    const serviceFileName = serviceToFileName(service.tagName)
    zhCN[serviceFileName] = service.description
    enUS[serviceFileName] = service.modelName + ' Management'
  }

  return { zhCN, enUS }
}

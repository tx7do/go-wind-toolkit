/**
 * Vue Vben - 路由代码生成器
 * 生成 router/routes/modules/app/*.ts 文件
 * 使用 RouteRecordRaw + BasicLayout + $t
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toPascalCase, toCamelCase } from '../../utils/openapi-parser'

export interface VbenRouterTemplateOptions {
  services: ParsedService[]
  moduleKey: string
  moduleDisplayName: string
  moduleIcon?: string
  moduleOrder?: number
  authority?: string[]
  modulePathMap?: Record<string, string>
}

/**
 * 根据模型名推断 Lucide 图标
 */
function inferIcon(modelName: string): string {
  const lower = modelName.toLowerCase()
  const iconMap: Record<string, string> = {
    user: 'lucide:user',
    role: 'lucide:shield-user',
    permission: 'lucide:shield-ellipsis',
    menu: 'lucide:square-menu',
    api: 'lucide:route',
    org: 'lucide:building',
    position: 'lucide:briefcase',
    tenant: 'lucide:building-2',
    dict: 'lucide:library-big',
    file: 'lucide:file-search',
    task: 'lucide:list-todo',
    language: 'lucide:globe',
    log: 'lucide:scroll-text',
    message: 'lucide:message-square',
    policy: 'lucide:shield-x',
  }

  for (const [key, icon] of Object.entries(iconMap)) {
    if (lower.includes(key)) return icon
  }
  return 'lucide:folder'
}

/**
 * 生成路由文件
 */
export function generateVbenRouterCode(options: VbenRouterTemplateOptions): string {
  const {
    services,
    moduleKey,
    moduleDisplayName,
    moduleIcon,
    moduleOrder = 2001,
    authority = [],
    modulePathMap = {},
  } = options

  // 生成子路由
  const children = services.map((service, index) => {
    const modelPascal = toPascalCase(service.modelName)
    const modelCamel = toCamelCase(service.modelName)
    const fileName = service.kebabName.replace(/-/g, '')
    const modulePath = modulePathMap[fileName] || fileName
    const routeName = `${modelPascal}Management`
    const path = fileName.replace(/([A-Z])/g, '-$1').toLowerCase().replace(/^-/, '') + 's'

    const icon = inferIcon(service.modelName)
    const auth = authority.length > 0 ? `\n          authority: ${JSON.stringify(authority)},` : ''

    return `      {
        path: '${path}',
        name: '${routeName}',
        meta: {
          order: ${index + 1},
          icon: '${icon}',
          title: $t('menu.${moduleKey}.${modelCamel}'),${auth}
        },
        component: () => import('#/views/app/${modulePath}/index.vue'),
      },`
  }).join('\n\n')

  const parentIcon = moduleIcon || 'lucide:folder'
  const parentAuth = authority.length > 0 ? `\n      authority: ${JSON.stringify(authority)},` : ''
  const parentPath = '/' + moduleKey.replace(/([A-Z])/g, '-$1').toLowerCase().replace(/^-/, '')

  let code = `import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const ${moduleKey}: RouteRecordRaw[] = [
  {
    path: '${parentPath}',
    name: '${toPascalCase(moduleKey)}',
    component: BasicLayout,
    redirect: '${parentPath}/${services[0] ? services[0].kebabName.replace(/-/g, '').replace(/([A-Z])/g, '-$1').toLowerCase().replace(/^-/, '') + 's' : 'index'}',
    meta: {
      order: ${moduleOrder},
      icon: '${parentIcon}',
      title: $t('menu.${moduleKey}.moduleName'),
      keepAlive: true,${parentAuth}
    },
    children: [
${children}
    ],
  },
];

export default ${moduleKey};
`
  return code
}

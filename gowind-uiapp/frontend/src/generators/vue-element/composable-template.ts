/**
 * Vue3 Element Plus - Composable 层代码生成器
 * 生成 api/composables/*.ts 文件 (Vue Query hooks)
 */
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { toCamelCase, toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface ComposableTemplateOptions {
  service: ParsedService
  apiImportPrefix?: string
  coreImportPath?: string
  serviceName?: string
  vueQueryImportPath?: string
  queryClientImportPath?: string
}

/**
 * 生成 Composable 层代码 (Vue Query hooks)
 */
export function generateComposableCode(options: ComposableTemplateOptions): string {
  const {
    service,
    apiImportPrefix = '@/api/generated',
    coreImportPath = '@/core/transport/rest',
    serviceName = 'admin',
    vueQueryImportPath = '@tanstack/vue-query',
    queryClientImportPath = '@/plugins/vue-query',
  } = options

  const crudPaths = getCrudPaths(service)
  const hasList = !!crudPaths.list
  const hasGet = !!crudPaths.get
  const hasCreate = !!crudPaths.create
  const hasUpdate = !!crudPaths.update
  const hasDelete = !!crudPaths.delete

  const modelCamel = toCamelCase(service.modelName)
  const modelPascal = toPascalCase(service.modelName)
  const serviceNameLower = service.kebabName.replace(/-/g, '')

  let code = ''

  // imports from vue-query
  code += `import {\n`
  code += `  useMutation,\n`
  code += `  type UseMutationOptions,\n`
  code += `  useQuery,\n`
  code += `  type UseQueryOptions,\n`
  code += `} from "${vueQueryImportPath}";\n`

  // type imports from generated
  const typeImports: string[] = []
  if (hasList) typeImports.push(`type ${serviceNameLower}_List${modelPascal}Response`)
  if (hasGet) typeImports.push(`type ${serviceNameLower}_${modelPascal}`)
  if (hasDelete) typeImports.push(`type ${serviceNameLower}_Delete${modelPascal}Request`)
  if (typeImports.length > 0) {
    code += `import {\n${typeImports.map(t => `  ${t},`).join('\n')}\n} from "${apiImportPrefix}/${serviceName}/service/v1";\n`
  }

  code += `import { makeUpdateMask, type PaginationQuery } from "${coreImportPath}";\n`

  // function imports from service layer
  const funcImports: string[] = []
  if (hasList) funcImports.push(`list${modelPascal}s`)
  if (hasGet) funcImports.push(`get${modelPascal}`)
  if (hasCreate) funcImports.push(`create${modelPascal}`)
  if (hasUpdate) funcImports.push(`update${modelPascal}`)
  if (hasDelete) funcImports.push(`delete${modelPascal}`)
  code += `import {\n  ${funcImports.join(',\n  ')},\n} from "@/api/service/${service.kebabName}";\n`
  code += `import { queryClient } from "${queryClientImportPath}";\n\n`

  // useList hook
  if (hasList) {
    code += `export function useList${modelPascal}s(\n`
    code += `  query: PaginationQuery,\n`
    code += `  options?: UseQueryOptions<${serviceNameLower}_List${modelPascal}Response, Error>\n`
    code += `) {\n`
    code += `  return useQuery({\n`
    code += `    queryKey: ["list${modelPascal}s", query],\n`
    code += `    queryFn: () => list${modelPascal}s(query),\n`
    code += `    ...options,\n`
    code += `  });\n`
    code += `}\n\n`

    // fetchList function (imperative)
    code += `export async function fetchList${modelPascal}s(params: PaginationQuery) {\n`
    code += `  return queryClient.fetchQuery({\n`
    code += `    queryKey: ["list${modelPascal}s", params],\n`
    code += `    queryFn: () => list${modelPascal}s(params),\n`
    code += `    retry: 0,\n`
    code += `  });\n`
    code += `}\n`
  }

  // useGet hook
  if (hasGet) {
    code += `\nexport function useGet${modelPascal}(\n`
    code += `  req: ${serviceNameLower}_Get${modelPascal}Request,\n`
    code += `  options?: UseQueryOptions<${serviceNameLower}_${modelPascal}, Error>\n`
    code += `) {\n`
    code += `  return useQuery({\n`
    code += `    queryKey: ["get${modelPascal}", req],\n`
    code += `    queryFn: () => get${modelPascal}(req),\n`
    code += `    ...options,\n`
    code += `  });\n`
    code += `}\n`
  }

  // useCreate hook
  if (hasCreate) {
    code += `\nexport function useCreate${modelPascal}(options?: UseMutationOptions<{}, Error, Record<string, any>>) {\n`
    code += `  return useMutation({\n`
    code += `    mutationFn: (values) => create${modelPascal}({ data: { ...values } as any }),\n`
    code += `    ...options,\n`
    code += `  });\n`
    code += `}\n`
  }

  // useUpdate hook
  if (hasUpdate) {
    code += `\nexport function useUpdate${modelPascal}(\n`
    code += `  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>\n`
    code += `) {\n`
    code += `  return useMutation({\n`
    code += `    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>\n`
    code += `      update${modelPascal}({\n`
    code += `        id,\n`
    code += `        data: { ...values } as any,\n`
    code += `        updateMask: makeUpdateMask(Object.keys(values ?? {})),\n`
    code += `      }),\n`
    code += `    ...options,\n`
    code += `  });\n`
    code += `}\n`
  }

  // useDelete hook
  if (hasDelete) {
    code += `\nexport function useDelete${modelPascal}(\n`
    code += `  options?: UseMutationOptions<{}, Error, ${serviceNameLower}_Delete${modelPascal}Request>\n`
    code += `) {\n`
    code += `  return useMutation({\n`
    code += `    mutationFn: (req) => delete${modelPascal}(req),\n`
    code += `    ...options,\n`
    code += `  });\n`
    code += `}\n`
  }

  return code
}

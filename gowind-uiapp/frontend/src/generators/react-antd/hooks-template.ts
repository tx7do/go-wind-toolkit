/**
 * React Ant Design Pro - React Query Hooks 代码生成器
 * 生成 api/hooks/*.ts 文件
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface ReactHooksTemplateOptions {
  service: ParsedService
  apiImportPrefix?: string
  coreImportPath?: string
  serviceName?: string
}

export function generateHooksCode(options: ReactHooksTemplateOptions): string {
  const {
    service,
    apiImportPrefix = '@/api/generated',
    coreImportPath = '@/core',
    serviceName = 'admin',
  } = options

  const crudPaths = getCrudPaths(service)
  const hasList = !!crudPaths.list
  const hasGet = !!crudPaths.get
  const hasCreate = !!crudPaths.create
  const hasUpdate = !!crudPaths.update
  const hasDelete = !!crudPaths.delete

  const modelPascal = toPascalCase(service.modelName)
  const modelCamel = service.modelCamelName
  const tagLower = service.kebabName.replace(/-/g, '')
  const fileName = service.kebabName.replace(/-/g, '')

  // Type imports from generated
  const typeImports: string[] = []
  if (hasCreate) typeImports.push(`type ${tagLower}_Create${modelPascal}Request`)
  if (hasDelete) typeImports.push(`type ${tagLower}_Delete${modelPascal}Request`)
  if (hasGet) typeImports.push(`type ${tagLower}_Get${modelPascal}Request`)
  if (hasList) typeImports.push(`type ${tagLower}_List${modelPascal}Response`)
  if (hasGet || hasList) typeImports.push(`type ${tagLower}_${modelPascal}`)

  // Function imports from service
  const funcImports: string[] = []
  if (hasList) funcImports.push(`list${modelPascal}s`)
  if (hasGet) funcImports.push(`get${modelPascal}`)
  if (hasCreate) funcImports.push(`create${modelPascal}`)
  if (hasUpdate) funcImports.push(`update${modelPascal}`)
  if (hasDelete) funcImports.push(`delete${modelPascal}`)

  let code = `import {\n  useMutation,\n  type UseMutationOptions,\n  useQuery,\n  type UseQueryOptions,\n} from '@tanstack/react-query';\n`

  if (typeImports.length > 0) {
    code += `import {\n${typeImports.map(i => `  ${i},`).join('\n')}\n} from '${apiImportPrefix}/${serviceName}/service/v1';\n`
  }

  code += `import { makeUpdateMask, type PaginationQuery, queryClient } from '${coreImportPath}';\n`
  code += `import { ${funcImports.join(', ')} } from '@/api/service/${fileName}';\n\n`

  // Section header
  code += `// ==============================\n// ${modelPascal} 管理\n// ==============================\n\n`

  // useList hook
  if (hasList) {
    code += `export function useList${modelPascal}s(\n  query: PaginationQuery,\n  options?: UseQueryOptions<${tagLower}_List${modelPascal}Response, Error>,\n) {\n  return useQuery({\n    queryKey: ['list${modelPascal}s', query],\n    queryFn: () => list${modelPascal}s(query),\n    ...options,\n  });\n}\n\n`

    // fetchList for ProTable request
    code += `export async function fetchList${modelPascal}s(params: PaginationQuery) {\n  return queryClient.fetchQuery({\n    queryKey: ['list${modelPascal}s', params],\n    queryFn: () => list${modelPascal}s(params),\n    retry: 0,\n  });\n}\n\n`
  }

  // useGet hook
  if (hasGet) {
    code += `export function useGet${modelPascal}(\n  req: ${tagLower}_Get${modelPascal}Request,\n  options?: UseQueryOptions<${tagLower}_${modelPascal}, Error>,\n) {\n  return useQuery({\n    queryKey: ['get${modelPascal}', req],\n    queryFn: () => get${modelPascal}(req),\n    ...options,\n  });\n}\n\n`
  }

  // useCreate hook
  if (hasCreate) {
    code += `export function useCreate${modelPascal}(\n  options?: UseMutationOptions<{}, Error, ${tagLower}_Create${modelPascal}Request>,\n) {\n  return useMutation({\n    mutationFn: (data) => create${modelPascal}(data),\n    ...options,\n  });\n}\n\n`
  }

  // useUpdate hook
  if (hasUpdate) {
    code += `export function useUpdate${modelPascal}(\n  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,\n) {\n  return useMutation({\n    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>\n      update${modelPascal}({\n        id,\n        data: { ...values } as any,\n        updateMask: makeUpdateMask(Object.keys(values ?? {})),\n      }),\n    ...options,\n  });\n}\n\n`
  }

  // useDelete hook
  if (hasDelete) {
    code += `export function useDelete${modelPascal}(\n  options?: UseMutationOptions<{}, Error, ${tagLower}_Delete${modelPascal}Request>,\n) {\n  return useMutation({\n    mutationFn: (req) => delete${modelPascal}(req),\n    ...options,\n  });\n}\n`
  }

  return code
}

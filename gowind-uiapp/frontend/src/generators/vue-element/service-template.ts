/**
 * Vue3 Element Plus - API Service 层代码生成器
 * 生成 api/service/*.ts 文件
 */
import type { ParsedService, ParsedField, ParsedOperation } from '../../utils/openapi-parser'
import { toCamelCase, toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface ServiceTemplateOptions {
  service: ParsedService
  /** API import 路径前缀，默认 @/api/generated */
  apiImportPrefix?: string
  /** core transport 路径，默认 @/core/transport/rest */
  coreImportPath?: string
  /** 服务名前缀（如 admin），用于 generated import 路径 */
  serviceName?: string
}

/**
 * 生成 API service 层代码
 */
export function generateServiceCode(options: ServiceTemplateOptions): string {
  const {
    service,
    apiImportPrefix = '@/api/generated',
    coreImportPath = '@/core/transport/rest',
    serviceName = 'admin',
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

  // 构建 import 列表
  const imports: string[] = []
  if (hasList) imports.push(`type ${serviceNameLower}_List${modelPascal}Response`)
  if (hasGet) imports.push(`type ${serviceNameLower}_${modelPascal}`)
  if (hasCreate) imports.push(`type ${serviceNameLower}_Create${modelPascal}Request`)
  if (hasUpdate) imports.push(`type ${serviceNameLower}_Update${modelPascal}Request`)
  if (hasDelete) imports.push(`type ${serviceNameLower}_Delete${modelPascal}Request`)

  const hasImports = imports.length > 0

  let code = ''
  code += hasImports
    ? `import {\n${imports.map(i => `  ${i},`).join('\n')}\n} from "${apiImportPrefix}/${serviceName}/service/v1";\n`
    : ''
  code += `import { type PaginationQuery, requestApi } from "${coreImportPath}";\n\n`

  // 生成 service client 单例
  code += `let _instance: ReturnType<typeof create${modelPascal}ServiceClient> | null = null;\n\n`
  code += `export function get${modelPascal}Service() {\n`
  code += `  if (!_instance) {\n`
  code += `    _instance = create${modelPascal}ServiceClient(requestApi);\n`
  code += `  }\n`
  code += `  return _instance;\n`
  code += `}\n`

  // list
  if (hasList) {
    code += `\nexport async function list${modelPascal}s(query: PaginationQuery) {\n`
    code += `  const params = query.toRawParams();\n`
    code += `  return get${modelPascal}Service().List(params);\n`
    code += `}\n`
  }

  // get
  if (hasGet) {
    code += `\nexport async function get${modelPascal}(request: ${serviceNameLower}_Get${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Get(request);\n`
    code += `}\n`
  }

  // create
  if (hasCreate) {
    code += `\nexport async function create${modelPascal}(request: ${serviceNameLower}_Create${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Create(request);\n`
    code += `}\n`
  }

  // update
  if (hasUpdate) {
    code += `\nexport async function update${modelPascal}(request: ${serviceNameLower}_Update${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Update(request);\n`
    code += `}\n`
  }

  // delete
  if (hasDelete) {
    code += `\nexport async function delete${modelPascal}(request: ${serviceNameLower}_Delete${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Delete(request);\n`
    code += `}\n`
  }

  return code
}

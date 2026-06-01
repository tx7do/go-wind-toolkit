/**
 * React Ant Design Pro - API Service 层代码生成器
 * 生成 api/service/*.ts 文件
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface ReactServiceTemplateOptions {
  service: ParsedService
  apiImportPrefix?: string
  coreImportPath?: string
  serviceName?: string
}

export function generateServiceCode(options: ReactServiceTemplateOptions): string {
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
  const tagLower = service.kebabName.replace(/-/g, '')

  const imports: string[] = []
  const funcName = `create${modelPascal}ServiceClient`
  imports.push(funcName)
  if (hasList) imports.push(`type ${tagLower}_List${modelPascal}Response`)
  if (hasGet) imports.push(`type ${tagLower}_Get${modelPascal}Request`)
  if (hasGet) imports.push(`type ${tagLower}_${modelPascal}`)
  if (hasCreate) imports.push(`type ${tagLower}_Create${modelPascal}Request`)
  if (hasUpdate) imports.push(`type ${tagLower}_Update${modelPascal}Request`)
  if (hasDelete) imports.push(`type ${tagLower}_Delete${modelPascal}Request`)

  let code = `import {\n${imports.map(i => `  ${i},`).join('\n')}\n} from '${apiImportPrefix}/${serviceName}/service/v1';\n`
  code += `import { type PaginationQuery, requestApi } from '${coreImportPath}';\n\n`

  code += `let _instance: ReturnType<typeof ${funcName}> | null = null;\n\n`
  code += `export function get${modelPascal}Service() {\n`
  code += `  if (!_instance) {\n`
  code += `    _instance = ${funcName}(requestApi);\n`
  code += `  }\n`
  code += `  return _instance;\n`
  code += `}\n`

  if (hasList) {
    code += `\nexport async function list${modelPascal}s(query: PaginationQuery) {\n`
    code += `  const params = query.toRawParams();\n`
    code += `  return get${modelPascal}Service().List(params);\n`
    code += `}\n`
  }
  if (hasGet) {
    code += `\nexport async function get${modelPascal}(request: ${tagLower}_Get${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Get(request);\n`
    code += `}\n`
  }
  if (hasCreate) {
    code += `\nexport async function create${modelPascal}(request: ${tagLower}_Create${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Create(request);\n`
    code += `}\n`
  }
  if (hasUpdate) {
    code += `\nexport async function update${modelPascal}(request: ${tagLower}_Update${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Update(request);\n`
    code += `}\n`
  }
  if (hasDelete) {
    code += `\nexport async function delete${modelPascal}(request: ${tagLower}_Delete${modelPascal}Request) {\n`
    code += `  return get${modelPascal}Service().Delete(request);\n`
    code += `}\n`
  }

  return code
}

/**
 * Vue Vben - API Service 层代码生成器
 * 生成 api/service/*.ts 文件
 * 使用 #/ 路径别名（Vben monorepo 约定）
 */
import type { ParsedService } from '../../utils/openapi-parser'
import { toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface VbenServiceTemplateOptions {
  service: ParsedService
  serviceName?: string
}

export function generateVbenServiceCode(options: VbenServiceTemplateOptions): string {
  const {
    service,
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
  if (hasCreate) imports.push(`type ${tagLower}_Create${modelPascal}Request`)
  if (hasDelete) imports.push(`type ${tagLower}_Delete${modelPascal}Request`)
  if (hasGet) imports.push(`type ${tagLower}_Get${modelPascal}Request`)
  if (hasUpdate) imports.push(`type ${tagLower}_Update${modelPascal}Request`)

  let code = `import {\n${imports.map(i => `  ${i},`).join('\n')}\n} from '#/api/generated/${serviceName}/service/v1';\n`
  code += `import { type PaginationQuery, requestApi } from '#/transport/rest';\n\n`

  code += `let _instance: null | ReturnType<typeof ${funcName}> = null;\n\n`
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
    code += `\nexport async function create${modelPascal}(\n  request: ${tagLower}_Create${modelPascal}Request,\n) {\n`
    code += `  return get${modelPascal}Service().Create(request);\n`
    code += `}\n`
  }
  if (hasUpdate) {
    code += `\nexport async function update${modelPascal}(\n  request: ${tagLower}_Update${modelPascal}Request,\n) {\n`
    code += `  return get${modelPascal}Service().Update(request);\n`
    code += `}\n`
  }
  if (hasDelete) {
    code += `\nexport async function delete${modelPascal}(\n  request: ${tagLower}_Delete${modelPascal}Request,\n) {\n`
    code += `  return get${modelPascal}Service().Delete(request);\n`
    code += `}\n`
  }

  return code
}

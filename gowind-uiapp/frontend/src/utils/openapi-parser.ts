/**
 * OpenAPI 3.0 YAML 解析器
 * 从 OpenAPI.yaml 中提取服务、模型、API端点等元数据
 */
import yaml from 'js-yaml'
import { toCamelCase, toPascalCase, toKebabCase } from './case-convert'

// ==============================
// 类型定义
// ==============================

export interface OpenApiSpec {
  openapi: string
  info: {
    title: string
    description?: string
    version: string
  }
  paths: Record<string, OpenApiPathItem>
  components?: {
    schemas: Record<string, OpenApiSchema>
  }
  tags?: OpenApiTag[]
}

export interface OpenApiTag {
  name: string
  description?: string
}

export interface OpenApiPathItem {
  [method: string]: OpenApiOperation
}

export interface OpenApiOperation {
  tags?: string[]
  operationId?: string
  description?: string
  parameters?: OpenApiParameter[]
  requestBody?: OpenApiRequestBody
  responses?: Record<string, OpenApiResponse>
}

export interface OpenApiParameter {
  name: string
  in: 'query' | 'path' | 'header'
  description?: string
  required?: boolean
  schema?: OpenApiSchema
}

export interface OpenApiRequestBody {
  content?: Record<string, {
    schema?: OpenApiSchema | { $ref: string }
  }>
  required?: boolean
}

export interface OpenApiResponse {
  description?: string
  content?: Record<string, {
    schema?: OpenApiSchema | { $ref: string }
  }>
}

export interface OpenApiSchema {
  type?: string
  format?: string
  description?: string
  properties?: Record<string, OpenApiSchemaProperty>
  items?: OpenApiSchema | { $ref: string }
  enum?: string[]
  required?: string[]
  $ref?: string
  example?: any
  readOnly?: boolean
}

export interface OpenApiSchemaProperty {
  type?: string
  format?: string
  description?: string
  enum?: string[]
  items?: OpenApiSchema | { $ref: string }
  $ref?: string
  example?: any
  readOnly?: boolean
}

// ==============================
// 解析后的业务模型
// ==============================

/** API操作类型 */
export type CrudOperation = 'list' | 'get' | 'create' | 'update' | 'delete' | 'other'

/** 解析后的服务信息 */
export interface ParsedService {
  /** 服务标签名（如 RoleService） */
  tagName: string
  /** 服务描述 */
  description: string
  /** kebab-case 服务路径名（如 role-service） */
  kebabName: string
  /** camelCase 服务名（如 roleService） */
  camelName: string
  /** PascalCase 服务名（如 RoleService） */
  pascalName: string
  /** 服务主模型名（如 Role） */
  modelName: string
  /** camelCase 模型名（如 role） */
  modelCamelName: string
  /** API路径前缀（如 /admin/v1/roles） */
  basePath: string
  /** 该服务支持的操作 */
  operations: ParsedOperation[]
  /** 主模型字段 */
  fields: ParsedField[]
}

/** 解析后的操作信息 */
export interface ParsedOperation {
  /** 操作类型 */
  type: CrudOperation
  /** HTTP方法 */
  method: string
  /** API路径 */
  path: string
  /** 操作描述 */
  description: string
  /** operationId */
  operationId: string
}

/** 解析后的字段信息 */
export interface ParsedField {
  /** 字段名 */
  name: string
  /** TypeScript类型 */
  tsType: string
  /** 描述 */
  description: string
  /** 是否是枚举 */
  isEnum: boolean
  /** 枚举值 */
  enumValues?: string[]
  /** OpenAPI格式 */
  format?: string
  /** 是否数组 */
  isArray: boolean
  /** 是否布尔 */
  isBoolean: boolean
  /** 是否日期 */
  isDate: boolean
  /** 是否整数 */
  isInteger: boolean
}

// ==============================
// 解析器
// ==============================

/**
 * 解析 OpenAPI YAML 字符串
 */
export function parseOpenApiYaml(yamlContent: string): OpenApiSpec {
  return yaml.load(yamlContent) as OpenApiSpec
}

/**
 * 从 OpenAPI 规范中提取所有 CRUD 服务
 */
export function extractServices(spec: OpenApiSpec): ParsedService[] {
  const tagMap = new Map<string, OpenApiTag>()
  if (spec.tags) {
    for (const tag of spec.tags) {
      tagMap.set(tag.name, tag)
    }
  }

  // 按 tag 分组 API 路径
  const tagOperations = new Map<string, ParsedOperation[]>()
  const tagBasePaths = new Map<string, string>()

  for (const [path, pathItem] of Object.entries(spec.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (['get', 'post', 'put', 'delete', 'patch'].includes(method) && operation.tags) {
        for (const tag of operation.tags) {
          if (!tagOperations.has(tag)) {
            tagOperations.set(tag, [])
          }
          tagOperations.get(tag)!.push({
            type: detectOperationType(method, path, operation),
            method: method.toUpperCase(),
            path,
            description: operation.description || '',
            operationId: operation.operationId || '',
          })

          // 记录基础路径（最长公共前缀，不含参数部分）
          const cleanPath = path.replace(/\/\{[^}]+\}$/, '')
          if (!tagBasePaths.has(tag) || cleanPath.length < tagBasePaths.get(tag)!.length) {
            tagBasePaths.set(tag, cleanPath)
          }
        }
      }
    }
  }

  // 构建服务列表
  const services: ParsedService[] = []
  for (const [tagName, operations] of tagOperations) {
    const tag = tagMap.get(tagName)
    const modelName = extractModelName(tagName)
    const modelCamelName = toCamelCase(modelName)
    const serviceBaseName = tagName.replace(/Service$/, '')

    // 从 schema 中提取模型字段
    const fields = extractModelFields(spec, modelName)

    services.push({
      tagName,
      description: tag?.description || tagName,
      kebabName: toKebabCase(serviceBaseName),
      camelName: toCamelCase(serviceBaseName),
      pascalName: serviceBaseName,
      modelName,
      modelCamelName,
      basePath: tagBasePaths.get(tagName) || '',
      operations,
      fields,
    })
  }

  return services
}

/**
 * 检测操作类型
 */
function detectOperationType(method: string, path: string, operation: OpenApiOperation): CrudOperation {
  const opId = (operation.operationId || '').toLowerCase()
  const desc = (operation.description || '').toLowerCase()

  // 通过 operationId 判断
  if (opId.includes('_list') || opId.endsWith('list')) return 'list'
  if (opId.includes('_get') || opId.endsWith('get')) return 'get'
  if (opId.includes('_create') || opId.endsWith('create')) return 'create'
  if (opId.includes('_update') || opId.endsWith('update')) return 'update'
  if (opId.includes('_delete') || opId.endsWith('delete')) return 'delete'

  // 通过描述判断
  if (desc.includes('列表') || desc.includes('查询') && method === 'get' && !path.includes('{')) return 'list'
  if (desc.includes('详情') || (method === 'get' && path.includes('{'))) return 'get'
  if (desc.includes('创建') || method === 'post') return 'create'
  if (desc.includes('更新') || method === 'put') return 'update'
  if (desc.includes('删除') || method === 'delete') return 'delete'

  return 'other'
}

/**
 * 从服务标签名提取模型名
 * 例如: RoleService -> Role, DictTypeService -> DictType
 */
function extractModelName(tagName: string): string {
  // 移除 Service 后缀
  let name = tagName.replace(/Service$/, '')

  // 特殊情况处理
  const specialCases: Record<string, string> = {
    'DictType': 'DictType',
    'DictEntry': 'DictEntry',
    'InternalMessageCategory': 'InternalMessageCategory',
    'InternalMessageRecipient': 'InternalMessage',
    'InternalMessage': 'InternalMessage',
    'DataAccessAuditLog': 'DataAccessAuditLog',
    'ApiAuditLog': 'ApiAuditLog',
    'LoginAuditLog': 'LoginAuditLog',
    'OperationAuditLog': 'OperationAuditLog',
    'PermissionAuditLog': 'PermissionAuditLog',
    'PolicyEvaluation': 'PolicyEvaluationLog',
    'AdminPortal': 'AdminPortal',
    'Authentication': 'Auth',
    'FileTransfer': 'FileTransfer',
    'UserProfile': 'UserProfile',
    'PermissionGroup': 'PermissionGroup',
    'OrgUnit': 'OrgUnit',
    'LoginPolicy': 'LoginPolicy',
  }

  return specialCases[name] || name
}

/**
 * 从 schema 定义中提取模型字段
 */
function extractModelFields(spec: OpenApiSpec, modelName: string): ParsedField[] {
  if (!spec.components?.schemas?.[modelName]) {
    // 尝试一些替代名称
    const altNames = [
      modelName,
      modelName + 'Response',
      'List' + modelName + 'Response',
    ]
    // 尝试在 schemas 中搜索
    for (const [schemaName, schema] of Object.entries(spec.components?.schemas || {})) {
      if (schemaName === modelName) {
        return schemaToFields(schema)
      }
    }
    return []
  }

  const schema = spec.components.schemas[modelName]
  return schemaToFields(schema)
}

/**
 * 将 OpenAPI Schema 转换为字段列表
 */
function schemaToFields(schema: OpenApiSchema): ParsedField[] {
  if (!schema.properties) return []

  const fields: ParsedField[] = []
  // 跳过的系统字段
  const skipFields = new Set(['createdBy', 'updatedBy', 'deletedBy', 'createdAt', 'updatedAt', 'deletedAt'])

  for (const [name, prop] of Object.entries(schema.properties)) {
    if (skipFields.has(name)) continue

    const resolved = resolveProperty(prop)
    fields.push({
      name,
      tsType: resolved.tsType,
      description: prop.description || name,
      isEnum: resolved.isEnum,
      enumValues: resolved.enumValues,
      format: prop.format,
      isArray: resolved.isArray,
      isBoolean: resolved.isBoolean,
      isDate: resolved.isDate,
      isInteger: resolved.isInteger,
    })
  }

  return fields
}

/**
 * 解析属性类型
 */
function resolveProperty(prop: OpenApiSchemaProperty): {
  tsType: string
  isEnum: boolean
  enumValues?: string[]
  isArray: boolean
  isBoolean: boolean
  isDate: boolean
  isInteger: boolean
} {
  const result = {
    tsType: 'string',
    isEnum: false,
    enumValues: undefined as string[] | undefined,
    isArray: false,
    isBoolean: false,
    isDate: false,
    isInteger: false,
  }

  if (prop.enum) {
    result.isEnum = true
    result.enumValues = prop.enum
    result.tsType = prop.enum.map(v => `'${v}'`).join(' | ')
    return result
  }

  if (prop.type === 'array') {
    result.isArray = true
    if (prop.items) {
      const itemType = 'type' in prop.items ? prop.items.type : undefined
      if (itemType === 'integer') {
        result.tsType = 'number[]'
        result.isInteger = true
      } else if (itemType === 'string') {
        result.tsType = 'string[]'
      } else {
        result.tsType = 'any[]'
      }
    } else {
      result.tsType = 'any[]'
    }
    return result
  }

  switch (prop.type) {
    case 'integer':
      result.tsType = 'number'
      result.isInteger = true
      break
    case 'boolean':
      result.tsType = 'boolean'
      result.isBoolean = true
      break
    case 'number':
      result.tsType = 'number'
      break
    case 'string':
      if (prop.format === 'date-time') {
        result.tsType = 'string'
        result.isDate = true
      } else {
        result.tsType = 'string'
      }
      break
    default:
      result.tsType = 'any'
  }

  return result
}

// ==============================
// 命名工具函数（re-export from case-convert）
// ==============================

export { toCamelCase, toPascalCase, toKebabCase } from './case-convert'

/**
 * 将 OpenAPI 中的 service tag 名转换为文件名
 * 例如: RoleService -> role
 *       DictTypeService -> dict-type
 */
export function serviceToFileName(tagName: string): string {
  const baseName = tagName.replace(/Service$/, '')
  return toKebabCase(baseName)
}

/**
 * 判断服务是否有完整的CRUD操作
 */
export function hasFullCrud(service: ParsedService): boolean {
  const opTypes = new Set(service.operations.map(op => op.type))
  return opTypes.has('list') && opTypes.has('create') && opTypes.has('update') && opTypes.has('delete')
}

/**
 * 获取服务的主要list/get/create/update/delete路径
 */
export function getCrudPaths(service: ParsedService): Record<CrudOperation, ParsedOperation | undefined> {
  const result: Record<CrudOperation, ParsedOperation | undefined> = {
    list: undefined,
    get: undefined,
    create: undefined,
    update: undefined,
    delete: undefined,
    other: undefined,
  }
  for (const op of service.operations) {
    if (!result[op.type]) {
      result[op.type] = op
    }
  }
  return result
}

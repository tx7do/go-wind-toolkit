/**
 * 通用命名风格转换工具函数
 * 提供 camelCase / PascalCase / snake_case / kebab-case 之间的互相转换
 */

/**
 * 将字符串首字母小写，转为 camelCase
 * 例如: "Role" -> "role", "RoleName" -> "roleName"
 */
export function toCamelCase(str: string): string {
  return str.charAt(0).toLowerCase() + str.slice(1)
}

/**
 * 将字符串首字母大写，转为 PascalCase
 * 例如: "role" -> "Role", "roleName" -> "RoleName"
 */
export function toPascalCase(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1)
}

/**
 * 将 camelCase / PascalCase 转为 snake_case
 * 例如: "roleName" -> "role_name", "sortOrder" -> "sort_order"
 */
export function toSnakeCase(str: string): string {
  return str.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`)
}

/**
 * 将 camelCase / PascalCase 转为 kebab-case
 * 例如: "RoleService" -> "role-service", "DictType" -> "dict-type"
 */
export function toKebabCase(str: string): string {
  return str
    .replace(/([A-Z])/g, '-$1')
    .toLowerCase()
    .replace(/^-/, '')
}

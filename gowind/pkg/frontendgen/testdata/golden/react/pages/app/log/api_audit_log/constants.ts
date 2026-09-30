/**
 * ApiAuditLog 模块常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

/** 请求方法 选项 */
export function getRequestMethodOptions(t: TFn) {
  return [
    { label: t('GET'), value: 'GET' },
    { label: t('POST'), value: 'POST' },
    { label: t('PUT'), value: 'PUT' },
    { label: t('DELETE'), value: 'DELETE' },
  ];
}

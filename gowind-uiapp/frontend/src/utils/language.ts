// 根据文件路径推断 Monaco 语言。
// 前端代码生成预览与 AI 后端生成计划预览共用。
export function detectLanguage(filePath: string): string {
  const base = filePath.split(/[\\/]/).pop() || ''
  const ext = base.includes('.') ? (base.split('.').pop()?.toLowerCase() || '') : ''
  const langMap: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    go: 'go',
    proto: 'proto',
    vue: 'html',
    html: 'html',
    css: 'css',
    scss: 'css',
    less: 'css',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    sql: 'sql',
    md: 'markdown',
    xml: 'xml',
  }
  return langMap[ext] || 'plaintext'
}

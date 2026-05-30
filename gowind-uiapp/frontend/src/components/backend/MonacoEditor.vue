<script setup lang="ts">
import {onMounted, ref, watch, onUnmounted} from 'vue'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'

// 注册 Web Worker（必须）
self.MonacoEnvironment = {
  getWorker(_: any, label: string) {
    return new editorWorker()
  }
}

const props = defineProps<{
  modelValue?: string
  dbType?: 'mysql' | 'postgresql' | 'sqlite' | 'oracle' // 新增数据库类型
  height?: string | number
  readOnly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'editorDidMount', editor: monaco.editor.IStandaloneCodeEditor): void
}>()

const editorRef = ref<HTMLDivElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

// 注册数据库特定的语法定义
function registerDatabaseLanguage(dbType: string) {
  // 卸载旧的语言定义（避免重复注册）
  try {
    monaco.languages.getLanguages().forEach(lang => {
      if (lang.id === 'sql-custom') {
        // Monaco 不支持直接卸载，通过覆盖方式处理
      }
    })
  } catch (e) {
    console.warn('清理语言定义时出错:', e)
  }

  // 定义不同数据库的关键字
  const keywords: Record<string, string[]> = {
    mysql: [
      'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT',
      'INNER', 'OUTER', 'ON', 'GROUP BY', 'ORDER BY', 'LIMIT', 'OFFSET', 'AS', 'DISTINCT',
      'CREATE', 'TABLE', 'DATABASE', 'INDEX', 'VIEW', 'TRIGGER', 'PROCEDURE', 'FUNCTION',
      'IF', 'EXISTS', 'NOT', 'NULL', 'AND', 'OR', 'BETWEEN', 'LIKE', 'IN', 'IS', 'CASE',
      'WHEN', 'THEN', 'ELSE', 'END', 'AUTO_INCREMENT', 'ENGINE', 'CHARSET', 'COLLATE',
      'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'INDEX', 'FULLTEXT', 'SPATIAL',
      'VARCHAR', 'TEXT', 'INT', 'BIGINT', 'FLOAT', 'DOUBLE', 'DECIMAL', 'DATE', 'DATETIME',
      'TIMESTAMP', 'JSON', 'ENUM', 'SET', 'BLOB', 'TINYINT', 'SMALLINT', 'MEDIUMINT',
      'YEAR', 'TIME', 'CHAR', 'BINARY', 'VARBINARY', 'TINYBLOB', 'MEDIUMBLOB', 'LONGBLOB',
      'TINYTEXT', 'MEDIUMTEXT', 'LONGTEXT', 'UNSIGNED', 'ZEROFILL', 'AUTO_INCREMENT'
    ],
    postgresql: [
      'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT',
      'INNER', 'OUTER', 'ON', 'GROUP BY', 'ORDER BY', 'LIMIT', 'OFFSET', 'AS', 'DISTINCT',
      'CREATE', 'TABLE', 'DATABASE', 'INDEX', 'VIEW', 'TRIGGER', 'FUNCTION', 'PROCEDURE',
      'IF', 'EXISTS', 'NOT', 'NULL', 'AND', 'OR', 'BETWEEN', 'LIKE', 'IN', 'IS', 'CASE',
      'WHEN', 'THEN', 'ELSE', 'END', 'SERIAL', 'BIGSERIAL', 'UUID', 'JSONB', 'JSON',
      'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'INDEX', 'CONCURRENTLY',
      'VARCHAR', 'TEXT', 'INT', 'BIGINT', 'FLOAT', 'DOUBLE PRECISION', 'NUMERIC', 'DECIMAL',
      'DATE', 'TIMESTAMP', 'TIMESTAMPTZ', 'INTERVAL', 'BOOLEAN', 'BYTEA', 'CIDR', 'INET',
      'MACADDR', 'ARRAY', 'RANGE', 'DOMAIN', 'TYPE', 'ENUM', 'SEQUENCE', 'GENERATED',
      'ALWAYS', 'STORED', 'VIRTUAL', 'PARTITION', 'RANGE', 'LIST', 'HASH', 'SUBPARTITION',
      'WINDOW', 'OVER', 'PARTITION BY', 'ROWS', 'RANGE', 'PRECEDING', 'FOLLOWING',
      'CURRENT ROW', 'UNBOUNDED', 'EXCLUDE', 'GROUPS', 'TIES', 'NO OTHERS'
    ],
    sqlite: [
      'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT',
      'INNER', 'OUTER', 'ON', 'GROUP BY', 'ORDER BY', 'LIMIT', 'OFFSET', 'AS', 'DISTINCT',
      'CREATE', 'TABLE', 'DATABASE', 'INDEX', 'VIEW', 'TRIGGER', 'IF', 'EXISTS', 'NOT',
      'NULL', 'AND', 'OR', 'BETWEEN', 'LIKE', 'GLOB', 'REGEXP', 'MATCH', 'IN', 'IS', 'CASE',
      'WHEN', 'THEN', 'ELSE', 'END', 'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE',
      'AUTOINCREMENT', 'INTEGER', 'TEXT', 'REAL', 'BLOB', 'NUMERIC', 'VACUUM', 'ANALYZE',
      'ATTACH', 'DETACH', 'REINDEX', 'PRAGMA', 'SAVEPOINT', 'RELEASE', 'ROLLBACK', 'COMMIT',
      'BEGIN', 'TRANSACTION', 'EXPLAIN', 'QUERY', 'PLAN', 'TEMP', 'TEMPORARY', 'WITHOUT',
      'ROWID', 'REPLACE', 'CONFLICT', 'ABORT', 'FAIL', 'IGNORE', 'ROLLBACK', 'RESTRICT',
      'CASCADE', 'SET', 'NULL', 'DEFAULT', 'CHECK', 'COLLATE', 'BINARY', 'NOCASE', 'RTRIM'
    ],
    oracle: [
      'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT',
      'INNER', 'OUTER', 'ON', 'GROUP BY', 'ORDER BY', 'FETCH', 'FIRST', 'ROWS', 'ONLY',
      'AS', 'DISTINCT', 'CREATE', 'TABLE', 'DATABASE', 'INDEX', 'VIEW', 'TRIGGER',
      'PROCEDURE', 'FUNCTION', 'PACKAGE', 'IF', 'EXISTS', 'NOT', 'NULL', 'AND', 'OR',
      'BETWEEN', 'LIKE', 'IN', 'IS', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'CONNECT BY',
      'START WITH', 'LEVEL', 'PRIOR', 'ROWNUM', 'ROWID', 'SYSDATE', 'DUAL', 'PRIMARY',
      'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'VARCHAR2', 'NVARCHAR2', 'NUMBER', 'DATE',
      'TIMESTAMP', 'INTERVAL', 'CLOB', 'BLOB', 'RAW', 'LONG', 'LONG RAW', 'BFILE', 'XMLTYPE',
      'ANYDATA', 'ANYDATASET', 'ANYTYPE', 'VARRAY', 'NESTED', 'TABLE', 'OBJECT', 'REF',
      'CURSOR', 'SYS_REFCURSOR', 'BULK', 'COLLECT', 'PARTITION', 'SUBPARTITION', 'ANALYZE',
      'COMMENT', 'GRANT', 'REVOKE', 'AUDIT', 'NOAUDIT', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
      'SET', 'TRANSACTION', 'ISOLATION', 'READ', 'WRITE', 'ONLY', 'FORCE', 'IMMEDIATE'
    ]
  }

  const selectedKeywords = keywords[dbType] || keywords.mysql

  // 注册自定义 SQL 语言（覆盖默认）
  monaco.languages.register({id: 'sql-custom'})

  // 配置语法高亮
  monaco.languages.setMonarchTokensProvider('sql-custom', {
    tokenizer: {
      root: [
        // 关键字（区分大小写）
        [new RegExp(`\\b(${selectedKeywords.join('|')})\\b`, 'i'), 'keyword'],
        // 字符串
        [/\'(?:[^\\']|\\.)*\'/, 'string'],
        [/\"(?:[^\\"]|\\.)*\"/, 'string'],
        // 注释
        [/--+.*/, 'comment'],
        [/\/\*/, 'comment', '@comment'],
        // 数字
        [/\d+/, 'number'],
        // 操作符
        [/[\+\-\*\/\%\=\!\>\<\&\|\^\~]/, 'operator'],
        // 标识符
        [/[\w_]+/, 'identifier']
      ],
      comment: [
        [/[^/*]+/, 'comment'],
        [/\*\//, 'comment', '@pop'],
        [/[\/*]/, 'comment']
      ]
    }
  })

  // 配置自动补全
  monaco.languages.registerCompletionItemProvider('sql-custom', {
    provideCompletionItems: (model, position) => {
      const suggestions: monaco.languages.CompletionItem[] = []

      // 添加关键字建议
      selectedKeywords.forEach(keyword => {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: keyword,
          range: {
            startLineNumber: position.lineNumber,
            startColumn: position.column,
            endLineNumber: position.lineNumber,
            endColumn: position.column
          }
        })
      })

      // 添加常用函数（根据数据库类型）
      const functions = getDatabaseFunctions(dbType)
      functions.forEach(fn => {
        suggestions.push({
          label: fn,
          kind: monaco.languages.CompletionItemKind.Function,
          insertText: fn.includes('(') ? fn : `${fn}($1)`,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range: {
            startLineNumber: position.lineNumber,
            startColumn: position.column,
            endLineNumber: position.lineNumber,
            endColumn: position.column
          }
        })
      })

      // 添加代码片段建议
      const snippets = getDatabaseSnippets(dbType)
      snippets.forEach((snippet: { label: any; snippet: any; description: any }) => {
        suggestions.push({
          label: snippet.label,
          kind: monaco.languages.CompletionItemKind.Snippet,
          insertText: snippet.snippet,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: snippet.description,
          range: {
            startLineNumber: position.lineNumber,
            startColumn: position.column,
            endLineNumber: position.lineNumber,
            endColumn: position.column
          }
        })
      })

      return {suggestions}
    }
  })


  return 'sql-custom'
}

// 获取数据库特定的函数列表
function getDatabaseFunctions(dbType: string): string[] {
  const functions: Record<string, string[]> = {
    mysql: [
      'COUNT()', 'SUM()', 'AVG()', 'MAX()', 'MIN()', 'CONCAT()', 'SUBSTRING()',
      'DATE_FORMAT()', 'NOW()', 'CURDATE()', 'CURTIME()', 'IF()', 'CASE',
      'GROUP_CONCAT()', 'JSON_EXTRACT()', 'JSON_OBJECT()', 'UUID()'
    ],
    postgresql: [
      'COUNT()', 'SUM()', 'AVG()', 'MAX()', 'MIN()', 'STRING_AGG()', 'ARRAY_AGG()',
      'TO_CHAR()', 'NOW()', 'CURRENT_DATE', 'CURRENT_TIME', 'CASE', 'COALESCE()',
      'NULLIF()', 'JSONB_EXTRACT_PATH()', 'JSONB_BUILD_OBJECT()', 'GENERATE_SERIES()',
      'ROW_NUMBER()', 'RANK()', 'DENSE_RANK()', 'LAG()', 'LEAD()'
    ],
    sqlite: [
      'COUNT()', 'SUM()', 'AVG()', 'MAX()', 'MIN()', 'GROUP_CONCAT()', 'SUBSTR()',
      'DATETIME()', 'DATE()', 'TIME()', 'JULIANDAY()', 'STRFTIME()', 'RANDOM()',
      'ABS()', 'ROUND()', 'UPPER()', 'LOWER()', 'LENGTH()', 'TRIM()', 'REPLACE()'
    ],
    oracle: [
      'COUNT()', 'SUM()', 'AVG()', 'MAX()', 'MIN()', 'LISTAGG()', 'TO_CHAR()',
      'TO_DATE()', 'SYSDATE', 'NVL()', 'DECODE()', 'CASE', 'ROWNUM', 'ROW_NUMBER()',
      'RANK()', 'DENSE_RANK()', 'LAG()', 'LEAD()', 'FIRST_VALUE()', 'LAST_VALUE()'
    ]
  }
  return functions[dbType] || functions.mysql
}

// 获取数据库特定的代码片段
function getDatabaseSnippets(dbType: string) {
  const snippets: Record<string, any[]> = {
    mysql: [
      {
        label: 'SELECT * FROM',
        snippet: 'SELECT * FROM `${1:table}` WHERE ${2:condition};',
        description: 'Select all columns'
      },
      {
        label: 'CREATE TABLE',
        snippet: 'CREATE TABLE `${1:table}` (\n\tid INT AUTO_INCREMENT PRIMARY KEY,\n\t${2:column} VARCHAR(255),\n\tcreated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;',
        description: 'Create table with MySQL defaults'
      }
    ],
    postgresql: [
      {
        label: 'SELECT * FROM',
        snippet: 'SELECT * FROM "${1:table}" WHERE ${2:condition};',
        description: 'Select all columns'
      },
      {
        label: 'CREATE TABLE',
        snippet: 'CREATE TABLE "${1:table}" (\n\tid SERIAL PRIMARY KEY,\n\t${2:column} VARCHAR(255),\n\tcreated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n);',
        description: 'Create table with PostgreSQL defaults'
      }
    ],
    sqlite: [
      {
        label: 'SELECT * FROM',
        snippet: 'SELECT * FROM "${1:table}" WHERE ${2:condition};',
        description: 'Select all columns'
      },
      {
        label: 'CREATE TABLE',
        snippet: 'CREATE TABLE "${1:table}" (\n\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t${2:column} TEXT,\n\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP\n);',
        description: 'Create table with SQLite defaults'
      }
    ],
    oracle: [
      {
        label: 'SELECT * FROM',
        snippet: 'SELECT * FROM "${1:table}" WHERE ${2:condition};',
        description: 'Select all columns'
      },
      {
        label: 'CREATE TABLE',
        snippet: 'CREATE TABLE "${1:table}" (\n\tid NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,\n\t${2:column} VARCHAR2(255),\n\tcreated_at TIMESTAMP DEFAULT SYSDATE\n);',
        description: 'Create table with Oracle defaults'
      }
    ]
  }
  return snippets[dbType] || snippets.mysql
}

onMounted(() => {
  if (!editorRef.value) return

  // 根据 dbType 注册语言
  const languageId = registerDatabaseLanguage(props.dbType || 'mysql')

  editor = monaco.editor.create(editorRef.value, {
    value: props.modelValue || '',
    language: languageId,
    theme: 'vs-light',
    readOnly: props.readOnly || false,
    automaticLayout: true,
    fontSize: 14,
    fontFamily: 'Consolas, "Courier New", monospace',
    minimap: {enabled: false},
    scrollBeyondLastLine: false,
    lineNumbers: 'on',
    roundedSelection: true,
    renderWhitespace: 'selection',
    cursorStyle: 'line',
    wordWrap: 'off',
    folding: true,
    showFoldingControls: 'mouseover',
    quickSuggestions: true,
    suggestOnTriggerCharacters: true,
    tabCompletion: 'on',
    selectionHighlight: true,
    occurrencesHighlight: 'multiFile',
    padding: {
      top: 10,
      bottom: 10
    },
    scrollbar: {
      vertical: 'auto',
      horizontal: 'auto',
      useShadows: false,
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8
    }
  })

  // 监听内容变化
  editor.onDidChangeModelContent(() => {
    const value = editor?.getValue() || ''
    emit('update:modelValue', value)
    emit('change', value)
  })

  emit('editorDidMount', editor)
})

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (editor && newVal !== editor.getValue()) {
    editor.setValue(newVal || '')
  }
})

// 监听数据库类型变化（重新配置编辑器）
watch(() => props.dbType, async (newType) => {
  if (!editor || !newType) return

  // 重新注册语言
  const languageId = registerDatabaseLanguage(newType)

  // 更新编辑器语言（需要重建模型）
  const model = editor.getModel()
  if (model) {
    monaco.editor.setModelLanguage(model, languageId)
  }
})

onUnmounted(() => {
  editor?.dispose()
})

// 暴露方法
defineExpose({
  getEditor: () => editor,
  getValue: () => editor?.getValue() || '',
  setValue: (value: string) => editor?.setValue(value),
  focus: () => editor?.focus(),
  formatDocument: () => {
    editor?.getAction('editor.action.formatDocument')?.run()
  }
})
</script>

<template>
  <div ref="editorRef" :style="{ height: typeof height === 'number' ? height + 'px' : height || '400px' }"></div>
</template>

<style scoped>
:deep(.monaco-editor) {
  border-radius: 4px;
  border: 1px solid #d9d9d9;
  transition: border-color 0.3s;
}

:deep(.monaco-editor.focused) {
  border-color: #4096ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}
</style>

export default {
  app: {
    title: 'GoWind Toolkit',
  },
  // Tab 标签
  tabs: {
    backend: '后端代码生成',
    frontend: '前端代码生成',
    remoteConfig: '远程配置',
  },
  // Header
  header: {
    switchLang: 'English',
  },
  // 通用
  common: {
    nextStep: '下一步',
    prevStep: '上一步',
    loading: '加载中...',
    importing: '正在导入...',
    refresh: '刷新',
    success: '成功',
    failed: '失败',
    reset: '重新开始',
    confirm: '确认',
    cancel: '取消',
    all: '全部',
  },
  // 后端代码生成页面
  backend: {
    // 步骤标题
    steps: {
      importSchema: '导入 Schema',
      tableConfig: '表配置',
      generateConfig: '生成配置',
    },
    // 项目
    project: {
      clickToOpen: '点击打开项目目录',
      selectGoProject: '选择 Go 微服务项目的根目录',
      identifying: '正在识别项目...',
      ready: '项目已就绪',
      switchProject: '切换项目',
      failed: '项目识别失败',
      hintGoMod: '请确保目录下包含 go.mod 文件',
      retry: '重新选择',
      noProject: '所选目录不是一个有效的 Go 项目，未找到 go.mod 文件',
      openFailed: '无法打开项目，请确认目录正确',
      services: '{count} 个服务',
      apiDefined: '已定义 API',
      apiNotDefined: '未定义 API',
    },
    // 导入 Schema
    import: {
      title: '导入 Schema',
      database: '数据库导入',
      file: '本地文件',
      remote: '远程地址',
      editor: 'SQL 编辑器',
      // 数据库
      dbType: '数据库类型',
      dsn: '数据源名称 (DSN)',
      dsnPlaceholder: '示例: mysql://user:password@localhost:3306/dbname?charset=utf8mb4',
      dsnRequired: '请输入数据源名称(DSN)',
      dsnMinLength: 'DSN长度至少5个字符',
      testConnection: '测试连接',
      importTables: '导入表结构',
      dbConnectSuccess: '数据库连接成功！',
      dbConnectFailed: '数据库连接失败',
      dbImportSuccess: '数据库导入成功',
      dbImportFailed: '数据库导入失败: {msg}',
      dbConfigError: '请检查数据库配置',
      // 文件
      fileDropHint: '点击或拖拽 SQL 文件到此处',
      fileFormatHint: '支持 .sql / .ddl 格式的 DDL 文件',
      fileDragWarning: '请拖入 .sql 或 .ddl 格式的文件',
      fileEmpty: '文件内容为空',
      fileReadFailed: '文件读取失败',
      sqlImportFailed: 'SQL 导入失败: {msg}',
      sqlFileImportSuccess: '从文件 {name} 成功导入表结构',
      // 远程
      remotePlaceholder: '输入 SQL DDL 文件 URL，如 https://example.com/schema.sql',
      fetchBtn: '拉取',
      remoteHint: '请输入可公开访问的 SQL DDL 文件地址',
      remoteUrlRequired: '请输入 SQL 文件地址',
      remoteEmpty: '获取到的内容为空',
      remoteFetchFailed: '拉取失败: {error}',
      remoteImportSuccess: '远程 SQL 导入成功',
      requestFailed: '请求失败: {status} {text}',
      // SQL 编辑器
      sqlPlaceholder: '请粘贴或输入 SQL DDL 语句（CREATE TABLE ...）...',
      importSql: '导入 SQL',
      openAdvancedEditor: '打开高级编辑器',
      sqlRequired: '请输入 SQL DDL 语句',
      sqlImportSuccess: 'SQL 导入成功',
      importFailed: '导入失败',
      networkError: '网络请求失败',
      // 已导入
      importedTables: '已导入 {count} 张表',
      nextStepConfig: '下一步：配置表',
      importSchemaFirst: '请先导入 Schema',
    },
    // 表配置
    table: {
      title: '表配置',
      tableCount: '{total} 张表，{excluded} 张已排除',
      appendImport: '追加导入',
      sqlImport: 'SQL 导入',
      tableName: '表名',
      service: '所属服务',
      selectService: '选择服务',
      quickSelect: '一键全选',
      exclude: '排除',
    },
    // 生成配置
    generate: {
      title: '生成目标',
      grpcService: 'gRPC 微服务',
      bffService: 'BFF 微服务',
      ormType: 'ORM 类型',
      bffServiceName: 'BFF 服务名',
      bffServiceNamePlaceholder: '如 admin',
      atLeastOne: '请至少选择一种生成目标',
      // 概览
      summary: '生成概览',
      project: '项目',
      validTables: '有效表数',
      genGrpc: '生成 gRPC',
      genBff: '生成 BFF',
      no: '否',
      startGenerate: '开始生成代码',
      grpcFailed: 'gRPC 代码生成失败: {msg}',
      grpcSuccess: 'gRPC 服务代码生成成功',
      bffFailed: 'BFF 代码生成失败: {msg}',
      bffSuccess: 'BFF 服务代码生成成功',
      codeGenFailed: '代码生成失败',
      nextStepGenerate: '下一步：生成配置',
    },
  },
  // 前端代码生成页面
  frontend: {
    // 步骤标题
    steps: {
      importOpenApi: '导入 OpenAPI',
      genConfig: '生成配置',
      previewGenerate: '预览 & 生成',
    },
    // 目标框架
    framework: {
      title: '目标框架',
      selectFramework: '目标框架',
    },
    // 导入 OpenAPI
    import: {
      title: '导入 OpenAPI 文档',
      local: '本地文件',
      remote: '远程地址',
      paste: '粘贴内容',
      // 本地文件
      fileDropHint: '点击或拖拽 OpenAPI 文件到此处',
      fileFormatHint: '支持 .yaml / .yml / .json 格式',
      fileDragWarning: '请拖入 .yaml / .yml / .json 格式的文件',
      // 远程
      remotePlaceholder: '输入 OpenAPI 文档 URL，如 https://petstore.swagger.io/v2/swagger.yaml',
      fetchBtn: '拉取',
      remoteHint: '请输入可公开访问的 OpenAPI 文档地址',
      remoteUrlRequired: '请输入 OpenAPI 文档地址',
      remoteEmpty: '获取到的内容为空',
      remoteFetchFailed: '拉取失败: {error}',
      remoteSuccess: '远程文档加载成功',
      requestFailed: '请求失败: {status} {text}',
      networkError: '网络请求失败',
      // 粘贴
      pastePlaceholder: '请粘贴 OpenAPI 3.0 YAML / JSON 内容...',
      parseBtn: '解析 OpenAPI',
      parseFailed: '解析 OpenAPI 失败: {msg}',
    },
    // 生成配置
    config: {
      title: '生成配置',
      outputDir: '输出目录',
      outputDirPlaceholder: '选择前端项目根目录',
      selectDir: '选择目录',
      generateTypes: '生成类型',
      serviceLayer: 'Service层',
      composableLayer: 'Composable层',
      listPage: '列表页面',
      editDrawer: '编辑抽屉',
      routerConfig: '路由配置',
      i18n: '国际化',
      notImplemented: '{framework} 代码生成器尚未实现，预览将显示占位内容',
    },
    // 服务选择
    service: {
      selectTitle: '选择要生成的服务 ({selected}/{total})',
      selectAll: '全选',
      deselectAll: '取消全选',
      selectListOnly: '仅选有列表的',
      fields: '{count} 个字段',
    },
    // 操作标签
    operation: {
      list: '列表',
      get: '详情',
      create: '创建',
      update: '更新',
      delete: '删除',
      other: '其他',
    },
    // 文件类型
    fileType: {
      all: '全部',
      page: '页面',
      drawer: '抽屉',
      router: '路由',
      locale: '国际化',
    },
    // 预览
    preview: {
      files: '文件 ({count})',
      noFiles: '没有生成的文件',
      previewBtn: '预览生成代码',
      resetBtn: '重新开始',
      confirmBtn: '确认生成代码',
    },
    // 占位
    placeholder: {
      notImplemented: '代码生成器尚未实现',
      service: '服务',
      description: '描述',
      fields: '字段数',
      operations: '操作',
      comingSoon: '敬请期待...',
    },
  },
}


# GoWind Toolkit

GoWind Toolkit 是一个基于 Go + Wails + Vue3 + Ant Design 的桌面端可视化开发工具，聚焦于数据库表结构解析、代码自动生成、远程配置等场景，助力后端与全栈开发者高效完成项目脚手架搭建与业务代码生成。

---

## 项目简介

本项目通过现代前后端分离架构，结合 Wails 框架实现跨平台桌面应用，前端采用 Vue3 + TypeScript + Ant Design Vue，后端基于 Go 实现数据库连接、表结构解析、代码生成等核心能力。

## 主要特性

- 支持 MySQL、PostgreSQL、SQLite 连接与表结构解析（Oracle 连接会明确报错，暂不可用）
- 代码生成的数据源是 MySQL / PostgreSQL 连接串与 DDL 文本，以及 `ent://` / `gorm://` Go 源码目录；SQLite 需先用 `sqlite3 <文件> .schema` 导出 DDL 再导入
- 一键导入 SQL 或数据库表，自动生成服务端/客户端/前端代码
- 支持 gRPC、RESTful、前端多种代码生成模板
- 可视化操作界面，便捷的表结构与服务配置
- 远程配置管理
- 跨平台支持（Windows、macOS、Linux）

## 技术栈

- 后端：Go 1.25+、Wails v2
- 前端：Vue 3、TypeScript、Vite、Ant Design Vue、Monaco Editor
- 依赖管理：Go Modules、pnpm/npm

## 命令行模式

同一可执行文件支持双模式：**不带参数**启动图形界面；**带命令行参数**进入无头 CLI 模式（项目探测、数据库、后端/前端代码生成、AI 助手、配置中心导出，JSON 输出、零交互，适合脚本与 AI Agent）。详见 [docs/cli.md](docs/cli.md)。

项目脚手架与开发工具（创建项目、添加服务、buf/ent/wire 生成）请使用 `gow` CLI，两者共享同一套生成库。

## 目录结构说明

```
gowind-uiapp/
├── app.go                # 应用主逻辑，Go 端核心业务
├── main.go               # 应用入口，Wails 启动配置
├── wails.json            # Wails 配置文件
├── go.mod/go.sum         # Go 依赖管理
├── internal/             # 后端核心模块
│   ├── database/         # 数据库连接与元数据解析
│   ├── detect/           # 项目检测与分析
│   └── generator/        # 代码生成逻辑
├── frontend/             # 前端源码
│   ├── src/
│   │   ├── App.vue       # 前端主入口
│   │   ├── components/   # 主要页面与功能组件
│   │   └── assets/       # 静态资源
│   ├── package.json      # 前端依赖与脚本
│   └── ...
├── cli/                  # 命令行模式（gowind-uiapp 带参数启动，非交互 JSON 输出，见 docs/cli.md）
├── docs/                 # 文档（cli.md 等）
├── build/                # 构建相关资源与平台适配
└── README.md             # 项目说明文档
```

## 安装与开发环境搭建

### 先决条件

- Go 1.25 及以上
- Node.js 16+、pnpm 或 npm
- Wails CLI（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）

### 安装依赖

```bash
# 安装 Go 依赖
go mod tidy

# 安装前端依赖
cd frontend
pnpm install   # 或 npm install
```

### 启动开发模式

```bash
# 启动热更新开发环境
wails dev
```

## 常用开发命令

```bash
# 前端开发
cd frontend
pnpm dev        # 启动 Vite 前端热更新
pnpm build      # 构建前端静态资源

# 后端/桌面端开发
wails dev       # 启动桌面端开发环境
wails build     # 构建生产包
```

## 构建与发布

```bash
# 一键打包跨平台桌面应用
wails build
```
构建产物位于 `build/bin/` 目录下，可直接分发。

## 贡献指南

欢迎社区开发者参与贡献！

1. Fork 本仓库并新建分支
2. 提交 Pull Request，描述变更内容
3. 遵循 Go 与前端代码规范，建议配合 issue 讨论

## 许可证

本项目采用 MIT License 开源协议。

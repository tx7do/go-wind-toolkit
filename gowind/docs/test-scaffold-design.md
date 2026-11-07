# CRUD 测试骨架生成——设计前置（暂缓实施）

状态：设计阶段（发现架构前置条件，未动工）
日期：2026-11-07

## 目标

后端生成管线（`pkg/sqlkratos`）产 service/data/proto，但不产测试。
为生成的 CRUD 服务补一套**可编译、可运行**的单测骨架。

## 发现的架构前置条件

1. **数据源没有接缝**：service 持有的是具体结构体指针
   （`*data.UserRepo`，service.tpl 的 `.DataSourceType`），不是接口。
   无法用"嵌入接口 + 覆写"的方式在测试里打桩——嵌入具体类型只会把
   nil 客户端带进来，一调就 panic。
2. **ent 的可测路径依赖额外驱动**：`enttest` + 内存 sqlite 是 ent 的
   惯用测法,但生成测试文件会引入 `github.com/mattn/go-sqlite3`
   （cgo）或纯 Go 驱动依赖——生成器往用户 go.mod 里塞依赖是不可接受的。

## 可选路线

| 路线 | 做法 | 代价 |
|---|---|---|
| A. 接口接缝 | 生成 repo 时同步产出 `Repo` 接口（或 service 侧声明消费者接口），测试骨架用 fake 实现 | 改 repo/ent_repo/gorm_repo 三个模板 + wiring；行为兼容但生成物变多 |
| B. 分层测试依赖 | 测试骨架放 `testdata`/`_test` 约定目录 + 生成独立的 `go_test.mod` | 工具链对多 go.mod 支持差,放弃 |
| C. 只生成编译安全的最小骨架 | 构造器冒烟 + CRUD 用例 `t.Skip` 占位 | 需要 bootstrap.Context 的测试构造方式验证,且测试价值低 |

## 结论

推荐路线 A：先在 repo 模板层加接口接缝（`data.Repo` 接口由 repo 实现类实现），
再产 `service_test.tpl`（fake repo + 表驱动 CRUD 用例,不引入新外部依赖）。
动工顺序：先给 sqlkratos 的一张 scratch 全量产物加接口改造 e2e,验证
wire/后处理链不回归,再上模板。

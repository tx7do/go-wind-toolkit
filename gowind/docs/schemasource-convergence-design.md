# sqlorm ↔ sqlproto 数据源层收敛设计

状态：设计稿（评审通过后动工）
日期：2026-11-07
范围：`gowind/pkg/sqlorm`、`gowind/pkg/sqlproto`、`gowind/pkg/sqlkratos`、`gowind/internal/schemasource`

## 1. 背景与现状

两条生成管线（SQL → ORM 模型、SQL → proto/gRPC/REST）在 2026-09 的收敛中已把
**数据源多路分发**上移到 `gowind/internal/schemasource`（mux、NormalizeDSN、text/goschema
驱动、IsJoinTable、SQLType 等），但仍有五处成对的重复逻辑留在两条管线内部。

当前依赖关系（单向）：

```
sqlkratos（组合根/编排器）
  ├──> sqlproto.Convert      （SQL → proto 定义）
  └──> sqlorm.Importer       （SQL → ent/gorm 模型）
sqlproto ──> internal/schemasource
sqlorm   ──> internal/schemasource
```

消费方：`gowind/internal/generate/cmd.go`（`gow generate`）与 gowind-uiapp
（GUI 向导、CLI backend 组）。对外签名不变是硬约束。

## 2. 已识别的重复（按收敛价值排序）

| # | 重复点 | 位置 | 收敛后归属 |
|---|---|---|---|
| R1 | include/exclude 表过滤循环，≥6 份 | entimport `mysql.go:40` / `postgres.go` / `text.go:191`；sqlproto `mysql.go:105` / `postgres.go:124` / `text.go:163` / `goschema.go:30` | `schemasource.FilterTables(tables, include, exclude)`，一处表驱动单测 |
| R2 | DDL 文本 → atlas `schema.Table` 构建双写（`toColumnType` unsigned 回溯逐行同构） | entimport `text.go`（399 行）vs sqlproto `internal/text.go`（202 行） | 上移 `schemasource.ParseDDLText(sql) (*schema.Schema, error)`（或并入 text 驱动 Inspector），两侧只留各自 sink |
| R3 | 方言分发 switch 骨架双写 | entimport `import.go NewImport` vs sqlproto `internal/convert.go NewConvert` | 参数化为「driver → inspect+filter → sink」管线（见 §3.1） |
| R4 | DSN 归一化第三份 | sqlkratos `generator.go:29 ensureDSNScheme`（`--driver` 补 scheme、PG key-value → URL） | 并入 `schemasource.NormalizeDSN`（增加 driver 提示参数），sqlkratos 改为透传 |
| R5 | 死代码与重复后处理 | sqlorm `internal/options.go`（`Options` 零引用）；entimport `formatSchemaFiles` 正则后处理 | 删除死代码；后处理下沉 `schemasource`（或 entimport 包内共享）仅供两处调用 |

## 3. 收敛方案

### 3.1 管线参数化（R3 的落地形态）

```go
// schemasource/pipeline.go（新）
// Sink 消费检查过的表清单；两条管线各自实现，转换逻辑留在管线内。
type Sink func(ctx context.Context, drv *Driver, tables []*schema.Table) error

func Inspect(ctx context.Context, dsn string,
    include, exclude []string, sink Sink) error {
    drv, err := Default.Open(ctx, dsn)   // 现有 mux
    if err != nil { return err }
    defer drv.Close()
    tables, err := drv.InspectSchema(ctx)  // goschema 驱动已含 include 过滤则幂等
    if err != nil { return err }
    tables, err = FilterTables(tables, include, exclude)  // R1
    if err != nil { return err }
    return sink(ctx, drv, tables)
}
```

- sqlproto 的 sink：`schema.Table` → `TableData/FieldData` → 模板渲染（proto 类型映射**留在 sqlproto**）。
- sqlorm 的 sink：`schema.Table` → ent Field 构建 / gorm gen（**留在 sqlorm**）。
- text 驱动特殊形态：text URL 直接得到 `*schema.Schema` 时，`Inspect` 短路（不再二次 inspect）。

### 3.2 语义分歧定案（不收敛，双方补注释）

1. **列类型映射两套宇宙**：`atlas 类型 → ent.Field` 与 `raw 类型 → proto 字符串`
   （含 sqlproto 特有的 `TINYINT(1)→bool`）。强行统一会造出双方都看不懂的中间类型。
2. **写盘机制**：schemast AST 打印 vs `text/template` 渲染，不可调和。
3. **`gorm://` 双语义**：sqlproto 侧是 mux 只读输入；sqlorm 侧 `ImporterFromGoSchema`
   直连 `ParseGormSchemaDir + BuildMySQLDDL`（绕过 mux）。解析器已共享，分发留在 sqlorm，
   在 `importer.go` 补注释说明为何不走 mux。
4. **sqlkratos 组合根地位**：wiring、多 server 装配、Kratos 分层是独有职责；
   收敛后其数据源职责只剩 `ensureDSNScheme → schemasource.NormalizeDSN` 的替换。

### 3.3 行为保持与验证

- 对外函数签名零变化（`Convert` / `Importer` / `Generate` 参数不动）。
- golden 保障：两管线现有全套件（entimport、sqlproto 渲染、sqlkratos 端到端）必须全绿；
  涉及产物字节的改动（R2 若影响列序/类型字符串）需先做逐字节 diff 验证再合入。
- `FilterTables` / `ParseDDLText` 上移时补表驱动单测（含 unsigned 回溯、serial、
  join table、include/exclude 大小写语义——以现有实现的并集为准，分歧点逐条记录）。

## 4. 实施顺序（每步独立可回滚）

1. ✅ **R1** FilterTables（2026-11-07 完成：6 处替换 + 8 例表驱动单测）
2. ✅ **R5** 死代码删除（sqlorm/internal/options.go 零引用已删；
   formatSchemaFiles 仅 entimport 单侧使用，评估后不上移）
3. ✅ **R4** NormalizeDSNWithDriver 吸收 ensureDSNScheme（driver 为空且
   未命中时原样放行的契约由测试钉死——gorm 分支接受面更宽）
4. ⬜ **R2** ParseDDLText 上移——**实施前必须先定基准语义**：两侧
   toColumnType 并非逐行同构，sqlproto 侧有两个承重扩展（unsigned 留在
   Raw 供类型字符串映射、PRIMARY KEY 强制 NOT NULL），entimport 侧没有。
   统一后需按设计门禁做两侧产物逐字节 diff 验证
5. ⬜ **R3** Inspect 管线参数化（最后做，前四步已把差异压到 sink 内）

## 5. 附带发现（顺手修）

- `sqlproto.Convert` 无条件 `MkdirAll(outputPath)`（converter.go:36），
  PlanTables 靠事后清理保证 dry-run 零副作用——建目录应挪到真正写盘的 render 层。
- entimport `TestMySQL` 为存量失败（凭据相关），已修（d0481ed），保持关注即可。

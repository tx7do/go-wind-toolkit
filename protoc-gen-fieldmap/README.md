# protoc-gen-fieldmap

从 proto 消息与**每适配器一份**的 YAML spec，生成强类型的字段映射函数，替代 go-wind 生态适配器中手工维护的"配置字段 → 上游选项"样板代码。

## 解决什么问题

[go-wind-bootstrap](https://github.com/tx7do/go-wind-bootstrap) 的每个适配器（`database/gorm`、`oss/minio`、`cache/redis`…）都有一段这样的手写映射：

```go
if driver := c.GetDriver(); driver != "" {
    options = append(options, gormCrud.WithDriverName(driver))
}
if v := c.GetMaxIdleConnections(); v > 0 {
    options = append(options, gormCrud.WithMaxIdleConns(int(v)))
}
```

上游每加一个配置项，就要走一遍"proto 加字段 → 适配器补映射 → 发版"的全流程，纯机械却占用了大量维护带宽。本生成器把这段劳动压缩为**声明一份 spec、跑一次生成**：

```yaml
message:
  proto: bootstrap/v1/database.proto
  type: Database.SQL
  func: MapSQLOptions
  target_type: gormCrud.Option
  target_qualifier: gormCrud
  fields:
    driver: { target: WithDriverName }
    max_idle_connections: { target: WithMaxIdleConns, cast: int, guard: positive }
    connection_max_lifetime_seconds: { target: WithConnMaxLifetime, cast: seconds }
```

生成（与手写代码逐行同风格，proto 注释自动作为函数文档）：

```go
func MapSQLOptions(c *v1.Database_SQL, opts []gormCrud.Option) []gormCrud.Option {
	if v := c.GetDriver(); v != "" {
		opts = append(opts, gormCrud.WithDriverName(v))
	}
	...
	if v := c.GetConnectionMaxLifetimeSeconds(); v != 0 {
		opts = append(opts, gormCrud.WithConnMaxLifetime(time.Duration(v)*time.Second))
	}
	return opts
}
```

spec 里写了 proto 上不存在的字段名、或对不兼容的类型用了 cast/guard，生成阶段直接报错——字段漂移在生成时被拦截，而不是在运行时。

## 用法

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-fieldmap@latest

protoc-gen-fieldmap -I conf/proto -spec specs -out .
```

| 参数 | 说明 |
|---|---|
| `-I` | proto 包含根，可重复；标准 WKT（google/protobuf/*）自动内建 |
| `-spec` | spec 文件或目录（目录取全部 `*.yaml`/`*.yml`，一文件一条规则），可重复 |
| `-out` | 输出根目录，默认 `.` |

> 虽沿用 `protoc-gen-*` 命名归入本仓库生成器家族，但它**不是 protoc 插件**：独立用 protocompile 读 proto 源码（含全部传递依赖），不依赖 protoc/buf 管线，适合挂 `go:generate` 或 CI 步骤。

## spec 参考

```yaml
output:
  file: database/gorm/mapping.gen.go   # 生成文件相对 -out 的路径
  package: gorm                        # 生成文件 Go 包名
imports:                               # 生成文件需要的导入：限定符 → 路径
  gormCrud: github.com/tx7do/go-crud/gorm
message:
  proto: bootstrap/v1/database.proto   # 相对 -I 的 proto 路径
  type: Database.SQL                   # 消息全名，可省略 proto 包前缀
  func: MapSQLOptions                  # 生成函数名
  style: option                        # option（默认）| struct
  target_type: gormCrud.Option         # option: 选项元素类型；struct: 目标结构体
  target_qualifier: gormCrud           # option 专用：字段 target 缺省限定符
  param: c                             # 参数名（默认 c）
  append: opts                         # option 风格的切片参数名（默认 opts）
  comment: ""                          # 函数文档；缺省取 proto 消息注释
  fields:
    driver: { target: WithDriverName }          # string → 判空卫语句（自动推导）
    migrate: { target: WithEnableMigrate }      # bool → 无条件应用（自动推导）
    max_idle_connections: { target: WithMaxIdleConns, cast: int, guard: positive }
    connection_max_lifetime_seconds: { target: WithConnMaxLifetime, cast: seconds }
    tls: { skip: true }                         # 显式记录"有意不映射"（如需手工逻辑）
```

### 三种风格

| style | 生成形态 | 适用场景 |
|---|---|---|
| `option`（默认） | `func Map(c, opts []T) []T`，逐字段卫语句 + `append` | 上游是 `With*` 选项函数（go-crud、wind-plugins 大多数） |
| `struct` | `func New(c) *T`，结构体字面量逐字段赋值，不保留卫语句 | 上游是配置结构体（minio `Config`、go-redis `UniversalOptions`） |
| `fill` | `func Fill(c, cfg *T) *T`，向既有配置对象逐字段覆盖赋值，保留卫语句 | 上游是"先构造后填字段"模式（registry 的 `api.DefaultConfig()` 等） |

fill 风格的被填充对象参数名用 `target_param` 指定（默认 `cfg`）：

```yaml
message:
  style: fill
  target_type: api.Config
  target_param: cfg
  fields:
    address: { target: Address }
```

### 传参展开（spread）

repeated 字段要传给变参选项函数时（如 `WithAddresses(addrs ...string)`），声明 `spread: true`，生成 `TARGET(v...)`。仅 option 风格、仅 list 字段。

### guard 词汇表（缺省 `auto` 按类型推导）

| guard | 条件 | 适用 |
|---|---|---|
| `auto` | 字符串判空、数值/枚举判零、布尔无条件、消息判 nil、集合/bytes 判长度 | 全部 |
| `none` | 无条件应用 | 全部 |
| `empty` | `v != ""` | 单值 string |
| `zero` | `v != 0`；bool 渲染为 `v`（true 才应用） | 单值数值/枚举/bool |
| `positive` | `v > 0` | 单值数值/枚举 |
| `not_nil` | `v != nil` | 单值 message |
| `len` | `len(v) > 0` | repeated / map / bytes |

### cast 词汇表（缺省不转换）

`int` `int32` `int64` `uint` `uint32` `uint64` `float32` `float64` `seconds`（数值秒 → `time.Duration`）`millis`（数值毫秒 → `time.Duration`）；枚举仅支持 `int`。`time` 导入按需自动补齐。

## 推荐工作流

- spec 与生成文件放在**适配器包内**（如 `database/gorm/mapping.spec.yaml` + `mapping.gen.go`），spec 是映射规则的唯一事实来源；
- `tls` 这类需要特殊处理的字段写 `skip: true`，留在 `client.go` 手工段；
- 上游加字段时：改 proto → spec 加一行 → 重跑生成 → golden/编译双保险；
- spec 是纯数据文件，天然适合交给 AI 代理按约定批量维护——人从"写映射"退到"review spec diff"。

## 测试

```bash
go test ./...            # golden 回归：生成器输出漂移即失败
go test -update ./...    # 刷新 golden
```

## License

MIT

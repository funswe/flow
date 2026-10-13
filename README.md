<div align="center">

# flow

**轻量级 Go Web 框架** —— 路由、中间件、参数绑定、ORM、Redis、JWT、HTTP 客户端、异步任务与定时器开箱即用

[![Go Reference](https://pkg.go.dev/badge/github.com/funswe/flow.svg)](https://pkg.go.dev/github.com/funswe/flow)
[![Go Report Card](https://goreportcard.com/badge/github.com/funswe/flow)](https://goreportcard.com/report/github.com/funswe/flow)
![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

</div>

基于 [httprouter](https://github.com/julienschmidt/httprouter) 构建，采用 Koajs 风格的洋葱圈中间件模型，内置 ORM、Redis、JWT、HTTP 客户端与结构化日志，适合快速搭建业务 API 服务。

## 特性

| 能力 | 说明 |
| --- | --- |
| 路由 | 基于 httprouter；支持 `GET/POST/PUT/PATCH/DELETE/HEAD/ALL`，注册的同时自动挂载同路径 `OPTIONS`，跨域预检开箱可用 |
| 中间件 | 洋葱圈模型，`next()` 前后均可执行逻辑；框架自带请求日志与跨域中间件 |
| 请求上下文 | 参数读取（字符串/整型/浮点/布尔，均带默认值版本）、`Parse` 参数绑定与必填校验、原始请求体、文件上传、客户端 IP / UA / Host / 协议 / Referer 等 |
| 响应 | 统一 JSON 响应、状态码、自定义响应头、重定向，以及自定义 `ResponseWriterAdapter` |
| 错误处理 | 统一 panic 处理器；处理器 panic 由框架在 handler 层收口，中间件仍能正常收尾，并可通过 `ctx.GetPanic()` 观测 |
| 可观测性 | 每个请求自动生成 `reqId` / `logId`，日志自动携带；`ctx.GetRequestId()` 可把业务记录与日志串联；`ResponseRecorder` 用于采集响应 |
| 生命周期 | `AddBefore` / `AddAfter` 钩子统一调度启动与收尾；收到信号时优雅停机 |
| 内置组件 | ORM（GORM）、Redis、JWT、HTTP 客户端（resty） |
| 异步能力 | 异步任务（`ExecuteTask` / `ExecuteAsyncTask`）与定时器（`StartTimer` / `StopTimer`） |
| 日志 | 基于 zap，按日期切割（rotatelogs），支持 JSON / 文本两种编码 |

## 环境要求

- Go 1.23 或更高版本

## 安装

```bash
go get -u github.com/funswe/flow
```

## 快速开始

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/funswe/flow"
)

func main() {
	flow.SetServerConfig(&flow.ServerConfig{AppName: "demo", Host: "0.0.0.0", Port: 8080})
	flow.SetLoggerConfig(&flow.LoggerConfig{LoggerLevel: "info", LoggerPath: "logs", LoggerMaxAge: 30})
	flow.SetJwtConfig(&flow.JwtConfig{Timeout: 24 * time.Hour, SecretKey: "please-change-me"})

	// 统一异常响应：处理器 panic 时由框架交给这里写出
	flow.SetPanicHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.Header().Set(flow.HttpHeaderContentType, "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":500,"msg":"系统异常"}`))
	})

	group := flow.NewRouterGroup()
	group.Use(func(ctx *flow.Context, next flow.Next) {
		ctx.Logger.Info("request begin")
		next()
		ctx.Logger.Info("request end")
	})
	group.POST("/demo/login", login)

	log.Fatal(flow.Run())
}

type loginParams struct {
	Account  string `json:"account" flow:"required:true"`
	Password string `json:"password" flow:"required:true"`
}

func login(ctx *flow.Context) {
	var params loginParams
	if err := ctx.Parse(&params); err != nil {
		ctx.Json(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}
	token, err := ctx.Jwt.Sign(map[string]interface{}{"account": params.Account})
	if err != nil {
		panic(err)
	}
	ctx.Json(map[string]interface{}{"code": 0, "msg": "success", "data": map[string]interface{}{"token": token}})
}
```

> 约定：响应体统一使用 `{"code":0,"msg":"success","data":{...}}` 信封，业务失败时用非 0 的 `code` 表达，HTTP 状态码保持 200。

## 配置

所有配置都在 `main` 中通过 `SetXxxConfig` 设置，未设置的字段使用默认值。

### ServerConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `AppName` | `string` | `flow` | 应用名称，同时作为日志文件名前缀 |
| `Proxy` | `bool` | `false` | 代理模式：开启后从 `X-Forwarded-Host` / `X-Forwarded-Proto` 获取真实 Host 与协议 |
| `Host` | `string` | `127.0.0.1` | 服务监听地址 |
| `Port` | `int` | `9505` | 服务监听端口 |

### LoggerConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `LoggerLevel` | `string` | `debug` | 日志级别：`debug` / `info` / `warn` / `error` / `dpanic` / `panic` / `fatal` |
| `LoggerPath` | `string` | 当前目录下 `logs` | 日志存放目录 |
| `LoggerMaxAge` | `int64` | `30` | 日志保留天数 |
| `FormatJson` | `bool` | `true` | 是否使用 JSON 编码 |

### OrmConfig / OrmPool

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `UserName` / `Password` | `string` | - | 数据库账号 |
| `DbName` | `string` | - | 数据库名 |
| `Host` / `Port` | `string` / `int` | - | 数据库地址 |
| `Pool` | `*OrmPool` | - | 连接池配置 |
| `Pool.MaxIdle` | `int` | `5` | 最大空闲连接数 |
| `Pool.MaxOpen` | `int` | `10` | 最大连接数 |
| `Pool.ConnMaxLifeTime` | `int` | `30000` | 连接最长存活时间 |
| `Pool.ConnMaxIdleTime` | `int` | `10000` | 连接最长空闲时间 |

### RedisConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `Enable` | `bool` | `false` | 是否启用 Redis |
| `Host` / `Port` | `string` / `int` | `127.0.0.1` / `6379` | Redis 地址 |
| `Password` | `string` | 空 | 密码 |
| `DbNum` | `int` | `0` | 库编号 |
| `Prefix` | `string` | `flow` | 键前缀，读写时自动拼接 |

### CorsConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `AllowOrigin` | `string` | `*` | 允许的来源 |
| `AllowedMethods` | `string` | `GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE` | 允许的方法 |
| `AllowedHeaders` | `string` | - | 允许的请求头 |

### CurlConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `Timeout` | `time.Duration` | `10s` | 请求超时 |
| `Headers` | `map[string]string` | - | 每个请求都会携带的头 |

### JwtConfig

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `Timeout` | `time.Duration` | `24h` | token 有效期 |
| `SecretKey` | `string` | 空 | 签名密钥，生产环境必须设置 |

## 路由与中间件

```go
// 方式一：全局默认分组
flow.Use(authMiddleware)
flow.POST("/admin/login", loginAction)

// 方式二：自建分组（推荐，便于按模块组织中间件）
group := flow.NewRouterGroup().
	Use(logMiddleware).
	Use(authMiddleware)
group.POST("/admin/book/list", bookListAction)
group.GET("/admin/book/detail", bookDetailAction)
```

- 中间件签名 `func(ctx *flow.Context, next flow.Next)`，调用 `next()` 之前的代码在处理器之前执行，之后的代码在处理器之后执行；
- 默认中间件依次为「请求日志」「跨域与 OPTIONS 预检」，自建分组的 `Use` 中间件排在其后；
- `OPTIONS` 预检由默认中间件直接应答，不进入业务处理器。

## 请求上下文

### 参数读取

```go
name := ctx.GetStringParam("name")            // 字符串
page := ctx.GetIntParamDefault("page", 1)     // 整型（缺省时用默认值）
id := ctx.GetInt64Param("id")                 // int64
price := ctx.GetFloat64Param("price")         // float64
enabled := ctx.GetBoolParam("enabled")        // bool
```

参数来源优先级：路由参数 → query / 表单 → JSON 请求体（同名时 JSON 覆盖前面的值）。

### 参数绑定与校验

```go
type createParams struct {
	Title   string `json:"title" flow:"required:true"`
	Sort    int    `json:"sort"`
}

var params createParams
if err := ctx.Parse(&params); err != nil {
	// 必填缺失时返回 *flow.FieldValidateError，可通过 err.(*flow.FieldValidateError) 取字段信息
	return
}
```

### 请求信息

| 方法 | 说明 |
| --- | --- |
| `ctx.GetMethod()` / `ctx.GetUri()` | 请求方法与路径 |
| `ctx.GetQuery()` / `ctx.GetQuerystring()` | query 参数（`url.Values` / 原始字符串） |
| `ctx.GetHeader(key)` / `ctx.GetHeaders()` | 请求头 |
| `ctx.GetHost()` / `ctx.GetHostname()` / `ctx.GetProtocol()` / `ctx.IsSecure()` | Host、hostname、协议 |
| `ctx.GetOrigin()` / `ctx.GetHref()` | 请求来源与完整链接 |
| `ctx.GetClientIp()` | 客户端 IP（`X-Forwarded-For` → `X-Real-Ip` → `RemoteAddr`） |
| `ctx.GetUserAgent()` / `ctx.GetLength()` | UA 与内容长度 |
| `ctx.GetRawBody()` / `ctx.GetRawStringBody()` | 原始请求体 |
| `ctx.FormFile(name)` | 上传文件（`*multipart.FileHeader`） |

### 上下文数据

```go
ctx.SetData("current_admin", admin)   // 并发安全，供中间件与处理器共享
value, exists := ctx.GetData("current_admin")
```

## 响应

```go
ctx.SetHeader("X-Trace-Id", traceId)
ctx.SetStatus(http.StatusForbidden)                 // 设置 HTTP 状态码
ctx.Json(map[string]interface{}{                     // 统一 JSON 响应
	"code": 403, "msg": "无权访问",
})
ctx.Redirect("/login", http.StatusFound)             // 重定向
```

自定义响应体可实现 `ResponseWriterAdapter`（`SetHeader(c *Context)` + `Data() ([]byte, error)`）后交给 `ctx.Res(adapter)`。

## 错误处理与 panic

```go
flow.SetPanicHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
	// 处理器 panic 的响应在这里统一写出（框架已记录完整堆栈）
})
```

- 处理器 panic 由框架在 **handler 外层**收口：中间件的 `next()` 之后仍会执行，可安全地做日志、指标等收尾；
- 中间件可通过 `ctx.GetPanic()` 拿到 panic 值与完整堆栈用于上报；
- 中间件自身的 panic 仍由路由层兜底，不影响其他请求。

## 可观测性

```go
// 1) 请求ID：与日志中的 reqId 一致，可用于把业务记录与日志关联
requestId := ctx.GetRequestId()

// 2) 上下文日志：自动携带 reqId、logId、ua
ctx.Logger.Info("business event", zap.String("orderId", orderId))

// 3) 响应采集：中间件可包装底层 writer，在 next() 之后读取状态码与响应体
recorder := flow.NewResponseRecorder(ctx.GetResWriter(), 4096) // 0 表示不限制响应体大小
ctx.SetResWriter(recorder)
next()
status, body := recorder.Status(), recorder.Body()
```

## 生命周期与优雅停机

```go
flow.AddBefore(func(app *flow.Application) { /* DB/Redis 初始化之后执行：连接、建索引等 */ })
flow.AddAfter(func(app *flow.Application) { /* 退出收尾：冲刷缓冲、断开连接等 */ })
flow.SetShutdownTimeout(5 * time.Second) // 可选，默认 5 秒

flow.Run()
```

- `AddBefore` 钩子在 DB / Redis 初始化完成后、开始监听之前按注册顺序执行；
- `AddAfter` 钩子在服务退出后**逆序**执行，适合「先冲刷缓冲、再断开连接」这类有依赖关系的收尾；
- 收到 `SIGINT` / `SIGTERM` 时优雅停机：停止接收新请求 → 等待在途请求结束（超过 `SetShutdownTimeout` 则强制关闭）→ 执行退出钩子 → 以 0 退出。

## 任务与定时器

```go
// 异步任务：实现 flow.Task 接口（Execute / Completed / Timeout 等）
flow.ExecuteTask(myTask)

// 异步聚合任务：同名任务在延迟窗口内会聚合，避免重复执行
flow.ExecuteAsyncTask(myTask)

// 定时器：实现 flow.Timer 接口（Run / GetInterval / IsPeriodic / IsImmediately）
flow.StartTimer(myTimer)
flow.StopTimer(myTimer.GetName())
```

## 内置组件

| 组件 | 入口 | 常用能力 |
| --- | --- | --- |
| ORM | `ctx.Orm.DB()` | 返回 `*gorm.DB`，可直接使用 GORM 全部能力；框架同时内置关系查询装配（`Model` / `Relation` / `SubRelation`） |
| Redis | `ctx.Redis` | `Get` / `Set` / `Delete` / `GetAllKeys` / `DeleteKeys` / `GetClient`，带前缀版本与 `WithOutPrefix` 版本 |
| JWT | `ctx.Jwt` | `Sign(map[string]interface{})` 签发、`Valid(token)` 校验并取回数据 |
| HTTP 客户端 | `ctx.Curl` | `Get(url, query, headers)` / `Post(url, body, headers)`，返回 `CurlResult` 可 `Parse(&v)` 解析 |

```go
// ORM
var admin models.Admin
if err := ctx.Orm.DB().Table(admin.TableName()).First(&admin, "uid = ?", uid).Error; err != nil {
	return err
}

// Redis
if err := ctx.Redis.Set("user:1", data, time.Hour); err != nil {
	return err
}
result, err := ctx.Redis.Get("user:1")
if err != nil && !ctx.Redis.IsNil(err) {
	return err
}

// HTTP 客户端
res, err := ctx.Curl.Post("https://example.com/api", map[string]string{"name": "flow"}, nil)
if err != nil {
	return err
}
var reply map[string]interface{}
if err := res.Parse(&reply); err != nil {
	return err
}
```

## 测试

```bash
go test ./...          # 全量测试
go test -cover ./...   # 带覆盖率
go vet ./...           # 静态检查
```

## 目录结构

```text
flow/
├── flow.go                # 对外 API：配置、路由注册、Run、任务与定时器
├── application.go         # Application：配置装配、启动流程、优雅停机
├── router.go              # 路由分组、中间件调度、默认中间件、panic 收口
├── context.go             # 请求上下文：参数、请求信息、响应、数据存取
├── request.go             # 请求封装
├── response.go            # 响应封装与 jsonWriter
├── response_recorder.go   # 响应采集器（供日志/监控中间件使用）
├── logger.go              # zap 日志（按日期切割）
├── orm.go                 # GORM 封装
├── redis.go               # Redis 封装
├── jwt.go                 # JWT 签发与校验
├── curl.go                # HTTP 客户端封装
├── task.go / timer.go     # 异步任务与定时器接口
└── utils/                 # 工具函数与 JSON 封装
```

## 版本变更

| 版本 | 变更 |
| --- | --- |
| `v1.10.9` | 新增 `Context.GetRequestId()`：返回与日志 `reqId` 一致的请求 ID，便于业务记录与日志串联 |
| `v1.10.8` | 新增优雅停机：收到 `SIGINT` / `SIGTERM` 后停止接收新请求并等待在途请求（默认 5 秒，可配），随后执行退出钩子并以 0 退出 |
| `v1.10.7` | 新增 `AddAfter` 退出钩子（逆序执行），资源初始化方可以自行挂载生命周期 |
| `v1.10.6` | panic 收口下移到 handler 层：中间件在 panic 后仍可正常收尾；新增 `Context.GetPanic()` |
| `v1.10.5` | 新增 `Context.GetStatus` / `GetResWriter` / `SetResWriter` 与 `ResponseRecorder` |

## 贡献

欢迎提交 Issue 与 Pull Request。

## License

[MIT](LICENSE) © 2019 funswe

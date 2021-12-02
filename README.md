# flow
flow是一个golang的web框架，使用koajs的洋葱圈中间件模型，框架内置了Orm，Redis，HttpClient，Jwt等工具，得益于[httprouter](https://github.com/julienschmidt/httprouter) ,性能提高30倍

# 安装
- go get -u github.com/funswe/flow

# Server配置
```
type ServerConfig struct {
	AppName    string // 应用名称，默认值flow
	Proxy      bool   // 是否是代理模式，默认值false
	Host       string // 服务启动地址，默认值127.0.0.1
	Port       int    // 服务端口，默认值9505
	ViewPath   string // 服务端渲染视图文件路径，使用pongo2模板，默认值当前目录下的views目录
	StaticPath string // 服务器静态资源路径，默认值当前目录下的statics
}
```
# Logger配置
日志使用的是[logrus](https://github.com/sirupsen/logrus) ，使用[rotatelogs](https://github.com/lestrrat-go/file-rotatelogs) 按日期分割日志
```
type LoggerConfig struct {
	LoggerLevel string // 日志级别，默认值debug
	LoggerPath  string // 日志存放目录，默认值当前目录下的logs
}
```
# Orm配置
orm框架使用的是[gorm](https://github.com/go-gorm/gorm) ，暂时只支持mysql
```
type OrmConfig struct {
	Enable   bool // 是否启用orm，默认值false
	UserName string // 数据库用户名
	Password string // 数据库密码
	DbName   string // 数据库名
	Host     string // 数据库地址，默认值127.0.0.1
	Port     int // 数据库端口，默认值3306
	Pool     *OrmPool // 数据库连接池相关配置
}

type OrmPool struct {
	MaxIdle         int // 连接池最大空闲链接，默认值5
	MaxOpen         int // 连接池最大连接数，默认值10
	ConnMaxLifeTime int64 // 连接最长存活期，超过这个时间连接将不再被复用，单位秒，默认值25000
	ConnMaxIdleTime int64 // 连接池里面的连接最大空闲时长，单位秒，默认值10
}
```
# Redis配置
redis使用的是[go-redis](https://github.com/go-redis/redis/v8)
```
type RedisConfig struct {
	Enable   bool // 是否启用redis，默认值false
	Password string // redis的密码
	DbNum    int // redis的库，默认值0
	Host     string // redis的地址，默认值127.0.0.1
	Port     int // redis的端口，默认值6379
	Prefix   string // redis的key前缀，默认值flow
}
```
# HttpClient配置
httpclient使用的是[go-resty](https://github.com/go-resty/resty/v2)
```
type CurlConfig struct {
	Timeout time.Duration     // 请求的超时时间，单位秒，默认值10
	Headers map[string]string // 统一请求的头信息
}
```
# Jwt配置
jwt使用的是[jwt-go](https://github.com/golang-jwt/jwt)
```
type JwtConfig struct {
	Timeout   time.Duration // 请求的超时时间，单位小时，默认值24
	SecretKey string        // 秘钥
}
```
# 跨域配置
```
type CorsConfig struct {
	Enable         bool // 是否开启跨域支持
	AllowOrigin    string // 跨域支持的域，默认值*
	AllowedHeaders string // 跨域支持的头
	AllowedMethods string // 跨域支持的请求方法，默认值GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE
}
```
# 示例
## 1、返回文本
```
func main() {
	flow.GET("/hello", func(ctx *flow.Context) {
		ctx.Body("hello, flow")
	})
	log.Fatal(flow.Run())
}
```
启动程序，在浏览器里访问http://localhost:9505/hello ,可以看到浏览器返回hello, flow
## 2、返回json
```
func main() {
	flow.GET("/json", func(ctx *flow.Context) {
		ctx.Json(map[string]interface{}{
			"msg": "hello, flow",
		})
	})
	log.Fatal(flow.Run())
}
```
启动程序，在浏览器里访问http://localhost:9505/json ,可以看到浏览器返回json字符串：{"msg": "hello, flow"}，Content-Type: application/json; charset=utf-8
## 3、获取请求参数
```
func main() {
	flow.GET("/param/:name", func(ctx *flow.Context) {
		name := ctx.GetStringParam("name")
		age := ctx.GetIntParam("age")
		ctx.Json(map[string]interface{}{
			"name": name,
			"age":  age,
		})
	})
	log.Fatal(flow.Run())
}
```
## 4、绑定参数
```
func main() {
	param := struct {
		Name string
		Age  int
	}{}
	flow.GET("/param/:name", func(ctx *flow.Context) {
		err := ctx.Parse(&param)
		if err != nil {
			panic(err)
		}
		ctx.Json(map[string]interface{}{
			"name": param.Name,
			"age":  param.Age,
		})
	})
	log.Fatal(flow.Run())
}
```
## 5、中间件使用
```
func main() {
	flow.Use(func(ctx *flow.Context, next flow.Next) {
		fmt.Println("mid1->start,time==", time.Now().UnixNano())
		next()
		fmt.Println("mid1->end,time===", time.Now().UnixNano())
	})
	flow.Use(func(ctx *flow.Context, next flow.Next) {
		fmt.Println("mid2->start,time==", time.Now().UnixNano())
		next()
		fmt.Println("mid2->end,time===", time.Now().UnixNano())
	})
	flow.GET("/middleware", func(ctx *flow.Context) {
		ctx.Body("middleware")
	})
	log.Fatal(flow.Run())
}
```
## 6、文件下载
```
func main() {
	flow.GET("/download", func(ctx *flow.Context) {
		ctx.Download("test-file.zip")
	})
	log.Fatal(flow.Run())
}
```

# [更多例子](https://github.com/funswe/flow-example)

# 模板
使用的HTML模板[pongo2](https://github.com/flosch/pongo2)





# flow
golang web frame as like koajs（洋葱圈模型）

# 安装
- go get -d github.com/funswe/flow
- go get -d github.com/julienschmidt/httprouter （路由处理：[httprouter](https://github.com/julienschmidt/httprouter)）

# 示例1
```
func main() {
	app := flow.New(false)
	app.POST("/hello", func(ctx *flow.Context) {
		ctx.Body("hello,world")
	})
	fmt.Println("启动...")
	app.Run(":12345")
}
```
启动程序，在浏览器里访问http://localhost:12345/test/hello，可以看到浏览器返回hello,world

# 示例2
```
func main() {
	app := flow.New(false)
	app.POST("/test/:name", func(ctx *flow.Context) {
		fmt.Println("name===", ctx.GetParam("name", ""))
		ctx.JsonResponse(map[string]interface{}{
			"name": ctx.GetParam("name", ""),
		})
	})
	app.Use(func(ctx *flow.Context, next flow.Next) {
		fmt.Println("mid1->start,time==", time.Now().UnixNano())
		next()
		fmt.Println("mid1->end,time===", time.Now().UnixNano())
	})
	fmt.Println("启动...")
	app.Run(":12345")
}
```
启动程序，在浏览器里访问http://localhost:12345/test/hello，可以看到浏览器返回{"name":"hello"}，终端打印
```
mid1->start,time== 1550045289462400763
name=== hello
mid1->end,time=== 1550045289462472332
```

# 示例3
```
type request struct {
	Age  int    `json:"age,string"`
	Name string `json:"name"`
}

func main() {
	app := flow.New(false)
	app.GET("/test/:name", func(ctx *flow.Context) {
       	req := &request{}
       	ctx.ParseStructure(req)
       	ctx.JsonResponse(map[string]interface{}{
       		"name": req.Name,
       		"age":  req.Age,
       	})
    }).POST("/test/:name", func(ctx *flow.Context) {
		req := &request{}
		ctx.ParseStructure(req)
		ctx.JsonResponse(map[string]interface{}{
			"name": req.Name,
			"age":  req.Age,
		})
	})
	fmt.Println("启动...")
	app.Run(":12345")
}
```
启动程序，在浏览器里访问http://localhost:12345/test/hello?age=30，可以看到浏览器返回{"age":30,"name":"hello"}

# API

## flow.New
创建一个app实例

## app.Run
启动服务

## app.Use
添加中间件

## app.GET
添加GET路由

## app.HEAD
添加HEAD路由

## app.OPTIONS
添加OPTIONS路由

## app.POST
添加POST路由

## app.PUT
添加PUT路由

## app.PATCH
添加PATCH路由

## app.DELETE
添加DELETE路由

## app.ALL
添加以上每种方法的路由

## app.StaticFiles
设置静态文件路径

## ctx.GetParam
获取请求参数，包括通配路由字段，json字段，query字段，form表单字段

## ctx.ParseStructure
将请求参数赋值到定义的结构体中，如示例3所示，方便管理请求数据

## ctx.GetHeaders
获取请求的所有头信息

## ctx.GetHeader
获取给定的头信息

## ctx.GetUri
获取请求的uri，如：http://localhost:12345/test/hello，返回/test/hello

## ctx.GetHost
获取请求的主机，如：http://localhost:12345/test/hello，返回localhost:12345

## ctx.GetProtocol
获取请求的协议类型，http|https

## ctx.IsSecure
判断是不是安全的连接，当Protocol是https返回true

## ctx.GetOrigin
获取请求的源，如：http://localhost:12345/test/hello，返回http://localhost:12345

## ctx.GetHref
获取请求的连接，如：http://localhost:12345/test/hello，返回http://localhost:12345/test/hello

## ctx.GetMethod
获取请求的方法

## ctx.GetQuery
获取请求的query参数，以map方式返回

## ctx.GetQuerystring
获取请求的query参数，以字符串方法返回

## ctx.GetHostname
获取请求的主机名，如：http://localhost:12345/test/hello，返回localhost

## ctx.GetLength
获取请求体的长度

## ctx.SetHeader
设置返回的头信息

## ctx.SetStatus
设置http返回码

## ctx.SetLength
设置返回体的长度

## ctx.Redirect
重定向

## ctx.Download
文件下载

## ctx.JsonResponse
以json方式返回

## ctx.Body
以文本方式返回



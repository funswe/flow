# flow
golang web frame as like koajs（洋葱圈模型）

# 安装
- go get -u github.com/funswe/flow

# 示例1
```
func main() {
	flow.ALL("/hello", func(ctx *flow.Context) {
		ctx.Body("hello,world")
	})
	fmt.Println("启动...")
	log.Fatal(flow.Run())
}
```
启动程序，在浏览器里访问http://localhost:9505/hello，可以看到浏览器返回hello,world

# 示例2
```
func main() {
	flow.GET("/test/:name", func(ctx *flow.Context) {
		fmt.Println("name===", ctx.GetParam("name"))
		ctx.Json(map[string]interface{}{
			"name": ctx.GetParam("name"),
		})
	})
	flow.Use(func(ctx *flow.Context, next flow.Next) {
		fmt.Println("mid1->start,time==", time.Now().UnixNano())
		next()
		fmt.Println("mid1->end,time===", time.Now().UnixNano())
	})
	fmt.Println("启动...")
	log.Fatal(flow.Run())
}
```
启动程序，在浏览器里访问http://localhost:9505/test/hello，可以看到浏览器返回{"name":"hello"}，终端打印
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
	flow.GET("/test/:name", func(ctx *flow.Context) {
       	req := &request{}
       	ctx.Parse(req)
       	ctx.Json(map[string]interface{}{
       		"name": req.Name,
       		"age":  req.Age,
       	})
    })
	fmt.Println("启动...")
	log.Fatal(flow.Run())
}
```
启动程序，在浏览器里访问http://localhost:9505/test/hello?age=30，可以看到浏览器返回{"age":30,"name":"hello"}

# [更多例子](https://github.com/funswe/flow-example)

# 模板
使用的HTML模板[pongo2](https://github.com/flosch/pongo2)





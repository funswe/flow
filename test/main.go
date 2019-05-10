package main

import (
	"fmt"
	"github.com/zhangmingfeng/flow"
)

type request struct {
	Age  int    `json:"age,string"`
	Name string `json:"name"`
}

func main() {
	app := flow.New(false)
	app.StaticFiles("files", "/tmp")
	app.ALL("/hello", func(ctx *flow.Context) {
		r := &request{}
		ctx.ParseStructure(r)
		fmt.Println(r)
		ctx.JsonResponse(map[string]interface{}{
			"aaa": ctx.GetQuery(),
			"bbb": ctx.GetHost(),
			"ccc": ctx.GetHostname(),
		})
		//ctx.Download("/tmp/sogou-qimpanel:0.pid")
		//ctx.Body("hello,world")
	})
	fmt.Println("启动...")
	app.Run(":12345")
}

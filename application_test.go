package flow

import (
	"testing"
	"fmt"
)

type ccc struct {
	C1 string `json:"c1"`
}

type appInfo struct {
	Aaa float64 `json:"aaa"`
	Bbb float64 `json:"bbb"`
	Ccc *ccc    `json:"ccc"`
}

func TestServer(t *testing.T) {
	app := New()
	fmt.Println("启动...")
	app.GET("/a/b/:name", func(ctx *Context) {
		ctx.SetStatus(404)
		ctx.JsonResponse(map[string]interface{}{
			"aaa":  111,
			"bbbb": 222,
			"data": ctx.params,
		})
	})
	app.POST("/a/b/:name", func(ctx *Context) {
		a := &appInfo{Ccc: &ccc{}}
		ctx.ParseStructure(a)
		fmt.Println("ccc===", ctx.GetParam("ccc", ""), ctx.GetParam("ddd", ""))
		ctx.JsonResponse(map[string]interface{}{
			"aaa":  111,
			"bbbb": 222,
			"data": ctx.params,
		})
	})
	app.Use(func(ctx *Context, next Next) {
		fmt.Println("mid1->start")
		next()
		fmt.Println("mid1->end")
	})
	app.Use(func(ctx *Context, next Next) {
		fmt.Println("mid2->start")
		next()
		fmt.Println("mid2->end")
	})
	app.Use(func(ctx *Context, next Next) {
		fmt.Println("mid3->start")
		//next()
		fmt.Println("mid3->end")
	})
	app.Use(func(ctx *Context, next Next) {
		fmt.Println("mid4->start")
		next()
		fmt.Println("mid4->end")
	})
	app.Use(func(ctx *Context, next Next) {
		fmt.Println("mid5->start")
		next()
		fmt.Println("mid5->end")
	})
	app.Run(":12345")
}

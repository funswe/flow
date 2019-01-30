package flow

import (
	"testing"
	"fmt"
)

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
		ctx.SetStatus(404)
		ctx.JsonResponse(map[string]interface{}{
			"aaa":  111,
			"bbbb": 222,
			"data": ctx.params,
		})
	})
	app.Use(func(ctx *Context) {
		fmt.Println("mid1")
	})
	app.Use(func(ctx *Context) {
		fmt.Println("mid2")
	})
	app.Use(func(ctx *Context) {
		fmt.Println("mid3")
	})
	app.Use(func(ctx *Context) {
		fmt.Println("mid4")
	})
	app.Use(func(ctx *Context) {
		fmt.Println("mid5")
	})
	app.Run(":12345")
}

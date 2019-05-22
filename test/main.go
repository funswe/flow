package main

import (
	"fmt"
	"github.com/funswe/flow"
)

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

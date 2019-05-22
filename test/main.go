package main

import (
	"fmt"
	"github.com/funswe/flow"
	"log"
)

type request struct {
	Age  int    `json:"age,string"`
	Name string `json:"name"`
}

func main() {
	app := flow.New()
	app.GET("/test/:name", func(ctx *flow.Context) {
		req := &request{}
		ctx.Parse(req)
		ctx.Json(map[string]interface{}{
			"name": req.Name,
			"age":  req.Age,
		})
	}).POST("/test/:name", func(ctx *flow.Context) {
		req := &request{}
		ctx.Parse(req)
		ctx.Json(map[string]interface{}{
			"name": req.Name,
			"age":  req.Age,
		})
	})
	fmt.Println("启动...")
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

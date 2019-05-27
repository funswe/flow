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
	flow.GET("/", func(ctx *flow.Context) {
		m := make(map[string]interface{})
		m["m"] = map[string]string{
			"body": "map-body",
		}
		m["obj"] = &request{
			Age:  12,
			Name: "zhangmingfeng",
		}
		ctx.Logger.Info(111)
		ctx.Render("index.html", m)
	})
	flow.GET("/test/:name", func(ctx *flow.Context) {
		req := &request{}
		ctx.Parse(req)
		ctx.Json(map[string]interface{}{
			"name": req.Name,
			"age":  req.Age,
		})
	})
	flow.POST("/test/:name", func(ctx *flow.Context) {
		req := &request{}
		ctx.Parse(req)
		ctx.Json(map[string]interface{}{
			"name": req.Name,
			"age":  req.Age,
		})
	})
	fmt.Println("启动...")
	err := flow.Run()
	if err != nil {
		log.Fatal(err)
	}
}

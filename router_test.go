package flow

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 中间件按注册顺序执行：默认中间件在前，Use 添加的在后，处理器最后
func TestMiddlewareExecutionOrder(t *testing.T) {
	var order []string
	rg := NewRouterGroup()
	rg.Use(func(ctx *Context, next Next) {
		order = append(order, "mw-1-before")
		next()
		order = append(order, "mw-1-after")
	})
	rg.Use(func(ctx *Context, next Next) {
		order = append(order, "mw-2")
		next()
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/unit/middleware-order", nil)
	handle(func(ctx *Context) { order = append(order, "handler") }, rg)(rec, req, nil)

	expected := []string{"mw-1-before", "mw-2", "handler", "mw-1-after"}
	if len(order) != len(expected) {
		t.Fatalf("执行链路不符：%v", order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("执行链路不符：%v，期望 %v", order, expected)
		}
	}
}

// 默认中间件写入跨域与 X-Powered-By 头
func TestDefaultMiddlewareHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/unit/default-headers", nil)
	handle(func(ctx *Context) {}, NewRouterGroup())(rec, req, nil)

	if got := rec.Header().Get(HttpHeaderXPoweredBy); got != "flow" {
		t.Fatalf("应写入 X-Powered-By: flow，实际 %q", got)
	}
	if got := rec.Header().Get(HttpHeaderCorsOrigin); got != app.corsConfig.AllowOrigin {
		t.Fatalf("跨域来源不符：%q", got)
	}
	if got := rec.Header().Get(HttpHeaderCorsMethods); len(got) == 0 {
		t.Fatal("应写入跨域允许的方法")
	}
}

// OPTIONS 预检请求由默认中间件直接应答，不进入业务处理器
func TestOptionsPreflightShortCircuitsHandler(t *testing.T) {
	handlerCalled := false
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/unit/preflight", nil)
	handle(func(ctx *Context) { handlerCalled = true }, NewRouterGroup())(rec, req, nil)

	if handlerCalled {
		t.Fatal("OPTIONS 预检不应进入处理器")
	}
	if body := rec.Body.String(); body != "true" {
		t.Fatalf("预检响应体应为 true，实际 %q", body)
	}
}

// 通过全局路由注册并分发请求：命中即执行处理器
func TestRouterRegistrationAndDispatch(t *testing.T) {
	GET("/unit/router/dispatch", func(ctx *Context) {
		ctx.Json(map[string]interface{}{"code": 0, "msg": "success"})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unit/router/dispatch", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为200，实际 %d", rec.Code)
	}
	if body := rec.Body.String(); body != `{"code":0,"msg":"success"}` {
		t.Fatalf("响应体不符：%s", body)
	}
}

// 未注册的路由走 NotFound 处理器
func TestRouterNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unit/router/not-registered", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("未注册路由应返回404，实际 %d", rec.Code)
	}
}

// 方法不匹配时不执行处理器
func TestRouterMethodNotMatched(t *testing.T) {
	handlerCalled := false
	GET("/unit/router/method", func(ctx *Context) { handlerCalled = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/unit/router/method", nil)
	router.ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("方法不匹配时不应执行处理器")
	}
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("方法不匹配应返回404或405，实际 %d", rec.Code)
	}
}

// Use 返回自身以支持链式调用
func TestRouterGroupUseChaining(t *testing.T) {
	rg := NewRouterGroup()
	before := len(rg.middleware)
	if got := rg.Use(func(ctx *Context, next Next) { next() }).Use(func(ctx *Context, next Next) { next() }); got != rg {
		t.Fatal("Use 应返回自身以支持链式调用")
	}
	if len(rg.middleware) != before+2 {
		t.Fatalf("中间件数量不符：%d", len(rg.middleware))
	}
}

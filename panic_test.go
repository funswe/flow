package flow

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// 处理器panic由flow在最内层收口：外层中间件能正常走完、堆栈完整、
// 响应仍交给应用配置的panic处理器写出（口径不变）。
func TestHandlerPanicCapturedInnermost(t *testing.T) {
	originalLogger := app.Logger
	app.Logger = zap.NewNop()
	defer func() { app.Logger = originalLogger }()

	// 模拟应用（如weims-server）的panic处理器：HTTP 200 + 业务码500
	SetPanicHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.Header().Set(HttpHeaderContentType, "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"code":500,"msg":%q}`, fmt.Sprint(err))))
	})
	defer SetPanicHandler(nil)

	var afterNext bool
	var gotValue interface{}
	var gotStack []byte

	rg := NewRouterGroup()
	rg.Use(func(ctx *Context, next Next) {
		next()
		// panic之后仍能执行，这正是访问日志等中间件需要的可观察性
		afterNext = true
		gotValue, gotStack, _ = ctx.GetPanic()
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/panic", nil)
	handle(func(ctx *Context) { panic("boom") }, rg)(rec, req, nil)

	if !afterNext {
		t.Fatal("panic之后外层中间件的next()后续代码应继续执行")
	}
	if gotValue != "boom" {
		t.Fatalf("GetPanic应返回panic值，实际 %#v", gotValue)
	}
	if !strings.Contains(string(gotStack), "panic_test.go") {
		t.Fatalf("堆栈应包含处理器帧，实际：\n%s", string(gotStack))
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("响应状态码应与应用panic处理器一致(200)，实际 %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"code":500`) || !strings.Contains(body, "boom") {
		t.Fatalf("响应体应为应用panic处理器写出的信封，实际：%s", body)
	}
}

// 未发生panic时，GetPanic应为空且不影响正常响应
func TestNoPanicWhenHandlerSucceeds(t *testing.T) {
	originalLogger := app.Logger
	app.Logger = zap.NewNop()
	defer func() { app.Logger = originalLogger }()

	var hasPanic bool
	var status int
	rg := NewRouterGroup()
	rg.Use(func(ctx *Context, next Next) {
		next()
		_, _, hasPanic = ctx.GetPanic()
		status = ctx.GetStatus()
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ok", nil)
	handle(func(ctx *Context) {
		ctx.SetStatus(http.StatusForbidden)
		ctx.Json(map[string]interface{}{"code": 403, "msg": "无权访问"})
	}, rg)(rec, req, nil)

	if hasPanic {
		t.Fatal("正常响应不应被标记为panic")
	}
	if status != http.StatusForbidden || rec.Code != http.StatusForbidden {
		t.Fatalf("状态码不符：ctx=%d rec=%d", status, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "无权访问") {
		t.Fatalf("响应体不符：%s", rec.Body.String())
	}
}

package flow

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GetRequestId 返回flow生成的请求ID，且与日志里的reqId是同一个值
func TestGetRequestIdMatchesLogReqId(t *testing.T) {
	// 日志writer(rotatelogs)会一直持有文件句柄，Windows下无法删除，因此用固定目录并尽力清理
	dir := filepath.Join(os.TempDir(), "flow-request-id-test")
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建日志目录失败：%v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	originalPath, originalFormat := app.loggerConfig.LoggerPath, app.loggerConfig.FormatJson
	app.loggerConfig.LoggerPath = dir
	app.loggerConfig.FormatJson = true
	defer func() {
		app.loggerConfig.LoggerPath = originalPath
		app.loggerConfig.FormatJson = originalFormat
	}()

	ctx := newContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/request-id", nil), nil, app)
	requestId := ctx.GetRequestId()
	if len(requestId) == 0 {
		t.Fatal("请求ID不应为空")
	}
	ctx.Logger.Info("probe")
	_ = ctx.Logger.Sync()

	logged := readLogDir(t, dir)
	if !strings.Contains(logged, `"reqId":"`+requestId+`"`) {
		t.Fatalf("日志中的reqId应与GetRequestId一致（%q），日志：%s", requestId, logged)
	}

	// 不同请求的ID不重复
	another := newContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/request-id", nil), nil, app)
	if another.GetRequestId() == requestId {
		t.Fatal("不同请求的ID不应重复")
	}
}

// 匿名上下文没有请求，取请求ID应返回空串且不panic
func TestGetRequestIdOnAnonymousContext(t *testing.T) {
	if id := NewAnonymousContext(app).GetRequestId(); id != "" {
		t.Fatalf("匿名上下文应返回空串，实际 %q", id)
	}
}

func readLogDir(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取日志目录失败：%v", err)
	}
	var content strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		content.Write(raw)
	}
	return content.String()
}

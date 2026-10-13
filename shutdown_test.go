package flow

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
)

// 优雅停机：Shutdown 之后 Serve 返回 ErrServerClosed
func TestShutdownStopsServer(t *testing.T) {
	originalLogger, originalTimeout := app.Logger, app.shutdownTimeout
	app.Logger = zap.NewNop()
	app.shutdownTimeout = 3 * time.Second
	defer func() { app.Logger, app.shutdownTimeout = originalLogger, originalTimeout }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.NewServeMux()}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()

	app.shutdown(srv)

	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("优雅停机后应返回 ErrServerClosed，实际 %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("优雅停机未生效")
	}
}

// 超时保护：在途请求长时间不结束时，按 shutdownTimeout 强制关闭
func TestShutdownForceCloseOnTimeout(t *testing.T) {
	originalLogger, originalTimeout := app.Logger, app.shutdownTimeout
	app.Logger = zap.NewNop()
	app.shutdownTimeout = 200 * time.Millisecond
	defer func() { app.Logger, app.shutdownTimeout = originalLogger, originalTimeout }()

	blocked := make(chan struct{})
	started := make(chan struct{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-blocked // 模拟慢请求
	})}
	go func() { _ = srv.Serve(ln) }()
	go func() { _, _ = http.Get("http://" + ln.Addr().String()) }()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("请求未进入处理器")
	}

	begin := time.Now()
	app.shutdown(srv)
	elapsed := time.Since(begin)
	close(blocked)

	if elapsed > 2*time.Second {
		t.Fatalf("超时保护未生效，shutdown 耗时 %v", elapsed)
	}
}

// 停机超时可配置，未配置时用默认值
func TestShutdownTimeoutConfigurable(t *testing.T) {
	original := app.shutdownTimeout
	defer func() { app.shutdownTimeout = original }()

	app.shutdownTimeout = 0
	if d := app.shutdownTimeoutOrDefault(); d != defShutdownTimeout {
		t.Fatalf("未配置时应使用默认值 %v，实际 %v", defShutdownTimeout, d)
	}
	SetShutdownTimeout(2 * time.Second)
	if d := app.shutdownTimeoutOrDefault(); d != 2*time.Second {
		t.Fatalf("配置后应生效，实际 %v", d)
	}
}

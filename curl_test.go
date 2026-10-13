package flow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestCurl(t *testing.T) *Curl {
	t.Helper()
	originalCurl, originalLogger := app.curlConfig, app.Logger
	app.curlConfig = defCurlConfig()
	app.curlConfig.Timeout = 2 * time.Second
	app.curlConfig.Headers = map[string]string{"X-Default": "flow"}
	app.Logger = zap.NewNop()
	t.Cleanup(func() {
		app.curlConfig, app.Logger = originalCurl, originalLogger
		app.Curl = nil
	})

	initCurl(app)
	if app.Curl == nil {
		t.Fatal("initCurl应初始化Curl对象")
	}
	return app.Curl
}

// Get：带query与自定义头，返回体可解析到结构体
func TestCurlGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HttpHeaderContentType, "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":          0,
			"query":         r.URL.Query().Get("keyword"),
			"defaultHeader": r.Header.Get("X-Default"),
			"customHeader":  r.Header.Get("X-Custom"),
			"method":        r.Method,
		})
	}))
	defer server.Close()

	curl := newTestCurl(t)
	res, err := curl.Get(server.URL, map[string]string{"keyword": "flow"}, map[string]string{"X-Custom": "1"})
	if err != nil {
		t.Fatalf("GET请求失败：%v", err)
	}
	if res.StatusCode() != http.StatusOK {
		t.Fatalf("状态码不符：%d", res.StatusCode())
	}

	var body map[string]interface{}
	if err := res.Parse(&body); err != nil {
		t.Fatalf("解析返回体失败：%v", err)
	}
	if body["query"] != "flow" || body["customHeader"] != "1" || body["defaultHeader"] != "flow" || body["method"] != http.MethodGet {
		t.Fatalf("请求参数不符：%v", body)
	}
}

// Post：JSON请求体被原样转发
func TestCurlPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.Header().Set(HttpHeaderContentType, "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"method": r.Method, "name": payload["name"]})
	}))
	defer server.Close()

	curl := newTestCurl(t)
	res, err := curl.Post(server.URL, map[string]interface{}{"name": "flow"}, nil)
	if err != nil {
		t.Fatalf("POST请求失败：%v", err)
	}

	var body map[string]interface{}
	if err := res.Parse(&body); err != nil {
		t.Fatalf("解析返回体失败：%v", err)
	}
	if body["method"] != http.MethodPost || body["name"] != "flow" {
		t.Fatalf("POST请求内容不符：%v", body)
	}
}

// 目标不可达时返回错误
func TestCurlRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	curl := newTestCurl(t)
	if _, err := curl.Get(url, nil, nil); err == nil {
		t.Fatal("目标不可达应返回错误")
	}
}

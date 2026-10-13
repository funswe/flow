package flow

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
)

const testJSONContentType = "application/json; charset=utf-8"

const formContentType = "application/x-www-form-urlencoded"

// useTempLogDir 把请求日志写进临时目录，避免测试污染仓库
func useTempLogDir(t *testing.T) {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "flow-test-logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建临时日志目录失败：%v", err)
	}
	original := app.loggerConfig.LoggerPath
	app.loggerConfig.LoggerPath = dir
	t.Cleanup(func() { app.loggerConfig.LoggerPath = original })
}

func newTestContext(t *testing.T, method, target string, body []byte, contentType string, params httprouter.Params) (*Context, *httptest.ResponseRecorder) {
	t.Helper()
	useTempLogDir(t)
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if len(contentType) > 0 {
		req.Header.Set(HttpHeaderContentType, contentType)
	}
	if len(body) > 0 {
		req.Header.Set(HttpHeaderContentLength, strconv.Itoa(len(body)))
	}
	rec := httptest.NewRecorder()
	return newContext(rec, req, params, app), rec
}

// JSON请求体与query都会被解析成参数，JSON优先级更高
func TestContextParamsFromJSONAndQuery(t *testing.T) {
	body := []byte(`{"name":"from-json","count":2,"ok":true,"price":1.5}`)
	ctx, _ := newTestContext(t, http.MethodPost, "/unit/ctx/params?name=from-query&page=3", body, testJSONContentType, nil)

	if got := ctx.GetStringParam("name"); got != "from-json" {
		t.Fatalf("JSON应覆盖query，实际 %q", got)
	}
	if got := ctx.GetIntParam("count"); got != 2 {
		t.Fatalf("count=2，实际 %d", got)
	}
	if got := ctx.GetBoolParam("ok"); !got {
		t.Fatal("ok应为true")
	}
	if got := ctx.GetFloat64Param("price"); got != 1.5 {
		t.Fatalf("price=1.5，实际 %v", got)
	}
	if got := ctx.GetIntParam("page"); got != 3 {
		t.Fatalf("query参数page=3，实际 %d", got)
	}
}

// 路由参数会被解析成参数
func TestContextParamsFromRoute(t *testing.T) {
	params := httprouter.Params{{Key: "id", Value: "42"}}
	ctx, _ := newTestContext(t, http.MethodGet, "/unit/ctx/route/42", nil, "", params)
	if got := ctx.GetInt64Param("id"); got != 42 {
		t.Fatalf("路由参数应生效，实际 %d", got)
	}
}

// 表单参数与默认值
func TestContextFormParamsAndDefaults(t *testing.T) {
	form := url.Values{"title": {"hello"}, "size": {"0"}}
	ctx, _ := newTestContext(t, http.MethodPost, "/unit/ctx/form", []byte(form.Encode()), formContentType, nil)

	if got := ctx.GetStringParam("title"); got != "hello" {
		t.Fatalf("表单参数不符：%q", got)
	}
	if got := ctx.GetStringParamDefault("missing", "def"); got != "def" {
		t.Fatalf("字符串默认值不符：%q", got)
	}
	if got := ctx.GetIntParamDefault("size", 9); got != 9 {
		t.Fatalf("数值为0时应回退默认值，实际 %d", got)
	}
	if got := ctx.GetInt64ParamDefault("missing", 8); got != 8 {
		t.Fatalf("int64默认值不符：%d", got)
	}
	if got := ctx.GetFloat64ParamDefault("missing", 6.5); got != 6.5 {
		t.Fatalf("float默认值不符：%v", got)
	}
	if got := ctx.GetBoolParamDefault("missing", true); !got {
		t.Fatal("bool默认值不符")
	}
}

type parseParams struct {
	Account string `json:"account" flow:"required:true"`
	Amount  int    `json:"amount"`
}

// Parse：缺失必填参数时返回字段校验错误
func TestContextParseRequiredField(t *testing.T) {
	ctx, _ := newTestContext(t, http.MethodPost, "/unit/ctx/parse", []byte(`{"amount":5}`), testJSONContentType, nil)

	err := ctx.Parse(&parseParams{})
	if err == nil {
		t.Fatal("缺失必填参数应返回错误")
	}
	fieldErr, ok := err.(*FieldValidateError)
	if !ok {
		t.Fatalf("应返回 *FieldValidateError，实际 %T", err)
	}
	if fieldErr.Param != "account" || fieldErr.Type != "required" {
		t.Fatalf("错误内容不符：%+v", fieldErr)
	}
	if !strings.Contains(fieldErr.Error(), "account") {
		t.Fatalf("错误信息应包含字段名：%s", fieldErr.Error())
	}
}

// Parse：参数齐全时赋值到目标对象
func TestContextParseSuccess(t *testing.T) {
	ctx, _ := newTestContext(t, http.MethodPost, "/unit/ctx/parse-ok", []byte(`{"account":"weims","amount":5}`), testJSONContentType, nil)

	var params parseParams
	if err := ctx.Parse(&params); err != nil {
		t.Fatalf("Parse失败：%v", err)
	}
	if params.Account != "weims" || params.Amount != 5 {
		t.Fatalf("解析结果不符：%+v", params)
	}
	if err := ctx.Parse(nil); err == nil {
		t.Fatal("nil对象应返回错误")
	}
}

// 请求信息：方法、URI、query、UA、长度与原始请求体
func TestContextRequestInfo(t *testing.T) {
	body := []byte(`{"a":1}`)
	ctx, _ := newTestContext(t, http.MethodPost, "/unit/ctx/info?x=1&y=2", body, testJSONContentType, nil)

	if got := ctx.GetMethod(); got != http.MethodPost {
		t.Fatalf("方法不符：%s", got)
	}
	if got := ctx.GetUri(); got != "/unit/ctx/info" {
		t.Fatalf("URI不符：%s", got)
	}
	if got := ctx.GetQuerystring(); got != "x=1&y=2" {
		t.Fatalf("querystring不符：%s", got)
	}
	if got := ctx.GetQuery().Get("y"); got != "2" {
		t.Fatalf("query解析不符：%s", got)
	}
	if got := ctx.GetLength(); got != len(body) {
		t.Fatalf("请求长度不符：%d", got)
	}
	raw, err := ctx.GetRawBody()
	if err != nil || string(raw) != string(body) {
		t.Fatalf("原始请求体不符：%s err=%v", raw, err)
	}
	if str, err := ctx.GetRawStringBody(); err != nil || str != string(body) {
		t.Fatalf("原始请求体字符串不符：%q err=%v", str, err)
	}
	if got := ctx.GetHeader(HttpHeaderContentType); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("请求头读取不符：%s", got)
	}
	if len(ctx.GetHeaders()) == 0 {
		t.Fatal("请求头集合不应为空")
	}
}

// 代理场景下的 host/协议/完整链接
func TestContextHostAndProtocolWithProxy(t *testing.T) {
	originalProxy := app.serverConfig.Proxy
	app.serverConfig.Proxy = true
	defer func() { app.serverConfig.Proxy = originalProxy }()

	useTempLogDir(t)
	req := httptest.NewRequest(http.MethodGet, "/unit/ctx/proxy?a=1", nil)
	req.Header.Set(HttpHeaderXForwardedHost, "api.example.com")
	req.Header.Set(HttpHeaderXForwardedProto, "https")
	req.Header.Set("User-Agent", "flow-test")
	ctx := newContext(httptest.NewRecorder(), req, nil, app)

	if got := ctx.GetHost(); got != "api.example.com" {
		t.Fatalf("代理场景host不符：%s", got)
	}
	if got := ctx.GetProtocol(); got != "https" {
		t.Fatalf("代理场景协议不符：%s", got)
	}
	if !ctx.IsSecure() {
		t.Fatal("代理场景应识别为https")
	}
	if got := ctx.GetOrigin(); got != "https://api.example.com" {
		t.Fatalf("origin不符：%s", got)
	}
	if got := ctx.GetHostname(); got != "api.example.com" {
		t.Fatalf("hostname不符：%s", got)
	}
	if got := ctx.GetHref(); got != "https://api.example.com/unit/ctx/proxy?a=1" {
		t.Fatalf("完整链接不符：%s", got)
	}
	if got := ctx.GetUserAgent(); got != "flow-test" {
		t.Fatalf("UA不符：%s", got)
	}
}

// 客户端IP优先取 X-Forwarded-For 的第一段，其次 X-Real-Ip，最后 RemoteAddr
func TestContextClientIp(t *testing.T) {
	useTempLogDir(t)

	req := httptest.NewRequest(http.MethodGet, "/unit/ctx/ip", nil)
	req.Header.Set(HttpHeaderXForwardedFor, "1.1.1.1, 2.2.2.2")
	if got := newContext(httptest.NewRecorder(), req, nil, app).GetClientIp(); got != "1.1.1.1" {
		t.Fatalf("X-Forwarded-For优先级不符：%s", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/unit/ctx/ip", nil)
	req.Header.Set(HttpHeaderXRealIp, "3.3.3.3")
	if got := newContext(httptest.NewRecorder(), req, nil, app).GetClientIp(); got != "3.3.3.3" {
		t.Fatalf("X-Real-Ip优先级不符：%s", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/unit/ctx/ip", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	if got := newContext(httptest.NewRecorder(), req, nil, app).GetClientIp(); got != "192.0.2.1" {
		t.Fatalf("RemoteAddr兜底不符：%s", got)
	}
}

// 上下文数据存取与并发安全
func TestContextDataMap(t *testing.T) {
	ctx, _ := newTestContext(t, http.MethodGet, "/unit/ctx/data", nil, "", nil)

	if _, exists := ctx.GetData("missing"); exists {
		t.Fatal("未设置的键不应存在")
	}
	ctx.SetData("current_admin", 123)
	value, exists := ctx.GetData("current_admin")
	if !exists || value.(int) != 123 {
		t.Fatalf("数据存取不符：%v %v", value, exists)
	}
}

// 响应设置：头、状态码、JSON、重定向与底层writer
func TestContextResponse(t *testing.T) {
	ctx, rec := newTestContext(t, http.MethodPost, "/unit/ctx/response", nil, "", nil)

	ctx.SetHeader(HttpHeaderContentType, testJSONContentType)
	ctx.SetHeader("X-Custom", "1")
	if got := rec.Header().Get("X-Custom"); got != "1" {
		t.Fatalf("响应头不符：%s", got)
	}

	ctx.SetStatus(http.StatusForbidden)
	if got := ctx.GetStatus(); got != http.StatusForbidden {
		t.Fatalf("状态码不符：%d", got)
	}

	ctx.Json(map[string]interface{}{"code": 403, "msg": "无权访问"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("响应状态码不符：%d", rec.Code)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("响应不是合法JSON：%s", rec.Body.String())
	}
	if envelope["msg"] != "无权访问" {
		t.Fatalf("响应内容不符：%v", envelope)
	}
	if got := rec.Header().Get(HttpHeaderContentType); got != testJSONContentType {
		t.Fatalf("JSON响应应写入Content-Type，实际 %q", got)
	}

	if ctx.GetResWriter() == nil {
		t.Fatal("应能获取底层ResponseWriter")
	}
	ctx.SetResWriter(rec)
	if ctx.GetResWriter() != rec {
		t.Fatal("SetResWriter应替换底层writer")
	}
}

// 重定向与响应长度
func TestContextRedirectAndLength(t *testing.T) {
	ctx, rec := newTestContext(t, http.MethodGet, "/unit/ctx/redirect", nil, "", nil)

	ctx.Redirect("/target", http.StatusFound)
	if rec.Code != http.StatusFound {
		t.Fatalf("重定向状态码不符：%d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/target" {
		t.Fatalf("重定向地址不符：%s", got)
	}

	ctx.SetLength(128)
	if got := rec.Header().Get(HttpHeaderContentLength); got != "128" {
		t.Fatalf("响应长度头不符：%s", got)
	}
}

// 上传文件：通过FormFile读取multipart表单中的文件
func TestContextFormFile(t *testing.T) {
	useTempLogDir(t)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("构造multipart失败：%v", err)
	}
	if _, err := part.Write([]byte("flow")); err != nil {
		t.Fatalf("写入multipart失败：%v", err)
	}
	_ = writer.WriteField("title", "上传标题")
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭multipart失败：%v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/unit/ctx/upload", &buf)
	req.Header.Set(HttpHeaderContentType, writer.FormDataContentType())
	ctx := newContext(httptest.NewRecorder(), req, nil, app)

	header, err := ctx.FormFile("file")
	if err != nil {
		t.Fatalf("读取上传文件失败：%v", err)
	}
	if header.Filename != "hello.txt" || header.Size != 4 {
		t.Fatalf("文件信息不符：%+v", header)
	}
	if got := ctx.GetStringParam("title"); got != "上传标题" {
		t.Fatalf("multipart普通字段不符：%q", got)
	}
}

// 匿名上下文没有请求对象，读取请求ID应为空且不panic
func TestAnonymousContextRequestId(t *testing.T) {
	ctx := NewAnonymousContext(app)
	if ctx == nil || ctx.GetApp() == nil {
		t.Fatal("匿名上下文应可用")
	}
	if got := ctx.GetRequestId(); got != "" {
		t.Fatalf("匿名上下文请求ID应为空，实际 %q", got)
	}
	if value, stack, ok := ctx.GetPanic(); ok || value != nil || stack != nil {
		t.Fatal("未发生panic时GetPanic应返回空值")
	}
}

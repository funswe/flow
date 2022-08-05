package flow

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/funswe/flow/utils/json"
)

type ResponseWriterAdapter interface {
	SetHeader(c *Context)
	Data() ([]byte, error)
}

// jsonWriter 返回json数据
type jsonWriter struct {
	data map[string]interface{}
}

func (j *jsonWriter) SetHeader(c *Context) {
	c.SetHeader(HttpHeaderContentType, "application/json; charset=utf-8")
}

func (j *jsonWriter) Data() ([]byte, error) {
	body, err := json.Marshal(j.data)
	return body, err
}

// 定义封装的response结构
type response struct {
	res http.ResponseWriter
	req *request
	app *Application
}

func newResponse(res http.ResponseWriter, req *request, app *Application) *response {
	return &response{res: res, req: req, app: app}
}

// 获取所有的返回头信息
func (r *response) getHeaders() map[string][]string {
	return r.res.Header()
}

// 获取返回头信息
func (r *response) getHeader(key string) string {
	return r.res.Header().Get(key)
}

// 设置返回头信息
func (r *response) setHeader(key, value string) *response {
	r.res.Header().Set(key, value)
	return r
}

// 设置http状态码
func (r *response) setStatus(code int) *response {
	r.res.WriteHeader(code)
	return r
}

// 设置返回内容的长度
func (r *response) setLength(length int) *response {
	r.setHeader(HttpHeaderContentLength, strconv.Itoa(length))
	return r
}

// 设置重定向地址
func (r *response) redirect(url string, code int) {
	http.Redirect(r.res, r.req.req, url, code)
}

// 下载文件
func (r *response) download(filePath string) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(app.serverConfig.StaticPath, filePath)
	}
	if _, err := os.Stat(filePath); err != nil {
		http.ServeFile(r.res, r.req.req, filePath)
		return
	}
	_, fileName := filepath.Split(filePath)
	r.setHeader(HttpHeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	r.setHeader(HttpHeaderContentType, "application/octet-stream")
	r.setHeader(HttpHeaderContentTransferEncoding, "binary")
	r.setHeader(HttpHeaderExpires, "0")
	r.setHeader(HttpHeaderCacheControl, "must-revalidate")
	http.ServeFile(r.res, r.req.req, filePath)
}

func (r *response) raw(data []byte) {
	if r.req.getMethod() != HttpMethodHead {
		r.res.Write(data)
	} else {
		r.res.Write([]byte{})
	}
}

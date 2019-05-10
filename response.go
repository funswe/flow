package flow

import (
	"fmt"
	"github.com/zhangmingfeng/flow/utils/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type response struct {
	app *Application
	res http.ResponseWriter
	req *http.Request
}

func newResponse(app *Application, res http.ResponseWriter, req *http.Request) *response {
	return &response{app, res, req}
}

func (r *response) getHeaders() map[string][]string {
	return r.res.Header()
}

func (r *response) getHeader(key string) string {
	return r.res.Header().Get(key)
}

func (r *response) setHeader(key, value string) *response {
	r.res.Header().Set(key, value)
	return r
}

func (r *response) setStatus(code int) *response {
	r.res.WriteHeader(code)
	return r
}

func (r *response) setLength(length int) *response {
	r.setHeader("Content-Length", strconv.Itoa(length))
	return r
}

func (r *response) redirect(url string, code int) {
	http.Redirect(r.res, r.req, url, code)
}

func (r *response) download(filePath string) {
	if _, err := os.Stat(filePath); err != nil {
		http.ServeFile(r.res, r.req, filePath)
		return
	}
	_, fileName := filepath.Split(filePath)
	r.setHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	r.setHeader("Content-Type", "application/octet-stream")
	r.setHeader("Content-Transfer-Encoding", "binary")
	r.setHeader("Expires", "0")
	r.setHeader("Cache-Control", "must-revalidate")
	http.ServeFile(r.res, r.req, filePath)
}

func (r *response) jsonResponse(data map[string]interface{}) {
	body, _ := json.Marshal(data)
	r.setHeader("Content-Type", "application/json; charset=utf-8")
	r.res.Write(body)
}

func (r *response) textResponse(data string) {
	r.setHeader("Content-Type", "text/plain; charset=utf-8")
	r.res.Write([]byte(data))
}

package flow

import (
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/funswe/flow/utils/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type rwresponse struct {
	http.ResponseWriter
	statusCode int
}

func (w *rwresponse) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

type response struct {
	res http.ResponseWriter
	req *http.Request
}

func newResponse(res http.ResponseWriter, req *http.Request) *response {
	return &response{res, req}
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

func (r *response) getStatusCode() int {
	b, ok := r.res.(*rwresponse)
	if ok {
		return b.statusCode
	}
	return 200
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

func (r *response) json(data map[string]interface{}) {
	body, _ := json.Marshal(data)
	r.setHeader("Content-Type", "application/json; charset=utf-8")
	r.res.Write(body)
}

func (r *response) text(data string) {
	r.setHeader("Content-Type", "text/plain; charset=utf-8")
	r.res.Write([]byte(data))
}

func (r *response) render(tmpFile string, data map[string]interface{}) {
	tpl, err := pongo2.FromCache(filepath.Join(viewPath, tmpFile))
	if err != nil {
		panic(err)
	}
	err = tpl.ExecuteWriter(data, r.res)
	if err != nil {
		panic(err)
	}
}

package flow

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/flosch/pongo2"
	"github.com/funswe/flow/utils/json"
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
	req *request
	app *Application
}

func newResponse(res http.ResponseWriter, req *request, app *Application) *response {
	return &response{res: res, req: req, app: app}
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
	r.setHeader(HttpHeaderContentLength, strconv.Itoa(length))
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
	http.Redirect(r.res, r.req.req, url, code)
}

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

func (r *response) json(data map[string]interface{}) {
	body, _ := json.Marshal(data)
	r.setHeader(HttpHeaderContentType, "application/json; charset=utf-8")
	r.raw(body)
}

func (r *response) text(data string) {
	r.setHeader(HttpHeaderContentType, "text/plain; charset=utf-8")
	r.raw([]byte(data))
}

func (r *response) render(tmpFile string, data map[string]interface{}) {
	tpl, err := pongo2.FromCache(filepath.Join(app.serverConfig.ViewPath, tmpFile))
	if err != nil {
		panic(err)
	}
	b, err := tpl.ExecuteBytes(data)
	//err = tpl.ExecuteWriter(data, r.res)
	if err != nil {
		panic(err)
	}
	r.raw(b)
}

func (r *response) raw(data []byte) {
	//etag := fmt.Sprintf("%x", sha1.Sum(data))
	//r.setHeader(HttpHeaderEtag, etag)
	if r.req.isFresh(r) {
		r.setStatus(304)
	}
	if r.getStatusCode() == 204 || r.getStatusCode() == 304 {
		r.res.Header().Del(HttpHeaderContentType)
		r.res.Header().Del(HttpHeaderContentLength)
		r.res.Header().Del(HttpHeaderTransferEncoding)
		data = []byte{}
	}
	if r.req.getMethod() != HttpMethodHead {
		r.res.Write(data)
	}
}

package flow

import (
	"net/http"
	"fmt"
	"net/url"
	"strings"
	"strconv"
)

type request struct {
	app *Application
	req *http.Request
}

func newRequest(app *Application, r *http.Request) *request {
	return &request{app, r}
}

func (r *request) getHeaders() map[string][]string {
	return r.req.Header
}

func (r *request) getHeader(key string) string {
	return r.req.Header.Get(key)
}

func (r *request) getUri() string {
	return r.req.URL.Path
}

func (r *request) getHost() string {
	proxy := r.app.GetProxy()
	var host string
	if proxy {
		host = r.getHeader("X-Forwarded-Host")
	}
	if len(host) == 0 {
		if r.req.ProtoMajor >= 2 {
			host = r.getHeader(":authority")
		}
		if len(host) == 0 {
			host = r.req.Host
		}
	}
	return host
}

func (r *request) getProtocol() string {
	if r.req.TLS != nil {
		return "https"
	}
	if !r.app.GetProxy() {
		return "http"
	}
	return r.getHeader("X-Forwarded-Proto")
}

func (r *request) isSecure() bool {
	return r.getProtocol() == "https"
}

func (r *request) getOrigin() string {
	return fmt.Sprintf("%s://%s", r.getProtocol(), r.getHost())
}

func (r *request) getHref() string {
	return fmt.Sprintf("%s%s", r.getOrigin(), r.req.RequestURI)
}

func (r *request) getMethod() string {
	return r.req.Method
}

func (r *request) getQuery() url.Values {
	return r.req.URL.Query()
}

func (r *request) getQuerystring() string {
	return r.req.URL.RequestURI()
}

func (r *request) getHostname() string {
	host := r.getHost()
	if len(host) == 0 {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		return r.req.URL.Hostname()
	}
	return strings.Split(host, ":")[0]
}

func (r *request) getLength() (l int) {
	length := r.getHeader("Content-Length")
	if len(length) == 0 {
		l = 0
		return
	}
	l, _ = strconv.Atoi(length)
	return
}

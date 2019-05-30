package flow

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type request struct {
	req *http.Request
	id  int64
}

func newRequest(r *http.Request, id int64) *request {
	return &request{r, id}
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
	if !proxy {
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
	return r.req.URL.RawQuery
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

func (r *request) isFresh(res *response) bool {
	method := r.getMethod()
	statusCode := res.getStatusCode()
	if method != "GET" && method != "HEAD" {
		return false
	}
	if (statusCode >= 200 && statusCode < 300) || statusCode == 304 {
		modifiedSince := r.getHeader("If-Modified-Since")
		noneMatch := r.getHeader("If-None-Match")
		if len(modifiedSince) == 0 && len(noneMatch) == 0 {
			return false
		}
		cacheControl := r.getHeader("Cache-Control")
		matched, _ := regexp.Match("(?:^|,)\\s*?no-cache\\s*?(?:,|$)", []byte(cacheControl))
		if len(cacheControl) > 0 && matched {
			return false
		}
		if len(noneMatch) > 0 && noneMatch != "*" {
			etag := res.getHeader("Etag")
			if len(etag) == 0 {
				return false
			}
		}
		if len(modifiedSince) > 0 {
			lastModified := res.getHeader("Last-Modified")
			if len(lastModified) == 0 {
				return false
			}
			lastModifiedTime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lastModified)
			if err != nil {
				return false
			}
			modifiedSinceTime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", modifiedSince)
			if err != nil {
				return false
			}
			if lastModifiedTime.After(modifiedSinceTime) {
				return false
			}
		}
		return true
	}
	return false
}

func (r *request) getUserAgent() string {
	return r.req.UserAgent()
}

package flow

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 定义封装的request结构
type request struct {
	req *http.Request // 原生的request对象
	id  int64         // 请求ID
	app *Application  // app对象
}

// 返回封装的request对象
func newRequest(r *http.Request, id int64, app *Application) *request {
	return &request{r, id, app}
}

// 获取所有的请求头信息
func (r *request) getHeaders() map[string][]string {
	return r.req.Header
}

// 获取请求头信息
func (r *request) getHeader(key string) string {
	return r.req.Header.Get(key)
}

// 获取请求的URI
func (r *request) getUri() string {
	return r.req.URL.Path
}

// 获取服务的HOST信息
func (r *request) getHost() string {
	var host string
	if app.serverConfig.Proxy {
		host = r.getHeader(HttpHeaderXForwardedHost)
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

// 获取请求的协议，http或者https
func (r *request) getProtocol() string {
	if r.req.TLS != nil {
		return "https"
	}
	if !app.serverConfig.Proxy {
		return "http"
	}
	return r.getHeader(HttpHeaderXForwardedProto)
}

// 判断请求是不是https
func (r *request) isSecure() bool {
	return r.getProtocol() == "https"
}

// 获取请求的地址，如http://www.demo.com
func (r *request) getOrigin() string {
	return fmt.Sprintf("%s://%s", r.getProtocol(), r.getHost())
}

// 获取请求完整链接，如http://www.demo.com/a/b
func (r *request) getHref() string {
	return fmt.Sprintf("%s%s", r.getOrigin(), r.req.RequestURI)
}

// 获取请求的方法，如GET，POST
func (r *request) getMethod() string {
	return r.req.Method
}

// 获取请求的query参数，map结构
func (r *request) getQuery() url.Values {
	return r.req.URL.Query()
}

// 获取请求的querystring
func (r *request) getQuerystring() string {
	return r.req.URL.RawQuery
}

// 获取请求的hostname信息
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

// 获取请求的内容长度
func (r *request) getLength() (l int) {
	length := r.getHeader(HttpHeaderContentLength)
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
	if method != HttpMethodGet && method != HttpMethodHead {
		return false
	}
	if (statusCode >= 200 && statusCode < 300) || statusCode == 304 {
		modifiedSince := r.getHeader(HttpHeaderIfModifiedSince)
		noneMatch := r.getHeader(HttpHeaderIfNoneMatch)
		if len(modifiedSince) == 0 && len(noneMatch) == 0 {
			return false
		}
		cacheControl := r.getHeader(HttpHeaderCacheControl)
		matched, _ := regexp.Match("(?:^|,)\\s*?no-cache\\s*?(?:,|$)", []byte(cacheControl))
		if len(cacheControl) > 0 && matched {
			return false
		}
		if len(noneMatch) > 0 && noneMatch != "*" {
			etag := res.getHeader(HttpHeaderEtag)
			if len(etag) == 0 {
				return false
			}
		}
		if len(modifiedSince) > 0 {
			lastModified := res.getHeader(HttpHeaderLastModified)
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

// 获取请求的UA
func (r *request) getUserAgent() string {
	return r.req.UserAgent()
}

// 获取请求的客户端的IP
func (r *request) getClientIp() string {
	xForwardedFor := r.getHeader(HttpHeaderXForwardedFor)
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if len(ip) != 0 {
		return ip
	}
	ip = strings.TrimSpace(r.getHeader(HttpHeaderXRealIp))
	if len(ip) != 0 {
		return ip
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

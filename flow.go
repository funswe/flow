package flow

import (
	"github.com/funswe/flow/log"
)

// 定义请求的方法
const (
	HttpMethodGet     = "GET"
	HttpMethodHead    = "HEAD"
	HttpMethodOptions = "OPTIONS"
	HttpMethodPost    = "POST"
	HttpMethodPut     = "PUT"
	HttpMethodPatch   = "PATCH"
	HttpMethodDelete  = "DELETE"
)

// 定义http头
const (
	HttpHeaderContentType             = "Content-Type"
	HttpHeaderContentLength           = "Content-Length"
	HttpHeaderTransferEncoding        = "Transfer-Encoding"
	HttpHeaderContentDisposition      = "Content-Disposition"
	HttpHeaderContentTransferEncoding = "Content-Transfer-Encoding"
	HttpHeaderExpires                 = "Expires"
	HttpHeaderCacheControl            = "Cache-Control"
	HttpHeaderEtag                    = "Etag"
	HttpHeaderXForwardedHost          = "X-Forwarded-Host"
	HttpHeaderXForwardedProto         = "X-Forwarded-Proto"
	HttpHeaderIfModifiedSince         = "If-Modified-Since"
	HttpHeaderIfNoneMatch             = "If-None-Match"
	HttpHeaderLastModified            = "Last-Modified"
	HttpHeaderXContentTypeOptions     = "X-Content-Type-Options"
	HttpHeaderXPoweredBy              = "X-Powered-By"
	HttpHeaderCorsOrigin              = "Access-Control-Allow-Origin"
	HttpHeaderCorsMethods             = "Access-Control-Allow-Methods"
	HttpHeaderCorsHeaders             = "Access-Control-Allow-Headers"
	HttpHeaderCorsMaxAge              = "Access-Control-Max-Age"
)

var (
	logFactory *log.Logger
	app        = &Application{
		rc:           make(chan int64),
		serverConfig: defServerConfig(),
		loggerConfig: defLoggerConfig(),
		ormConfig:    defOrmConfig(),
		redisConfig:  defRedisConfig(),
		corsConfig:   defCorsConfig(),
		curlConfig:   defCurlConfig(),
		jwtConfig:    defJwtConfig(),
		orm:          defOrm(),
		redis:        defRedis(),
		curl:         defCurl(),
		jwt:          defJwt(),
	}
	defRouterGroup = &RouterGroup{}
)

// 添加中间件
func Use(m Middleware) {
	defRouterGroup.Use(m)
}

// 设置服务配置
func SetServerConfig(serverConfig *ServerConfig) {
	if serverConfig == nil {
		serverConfig = defServerConfig()
	}
	app.setServerConfig(serverConfig)
}

// 设置日志配置
func SetLoggerConfig(loggerConfig *LoggerConfig) {
	if loggerConfig == nil {
		loggerConfig = defLoggerConfig()
	}
	app.setLoggerConfig(loggerConfig)
}

// 设置数据库配置
func SetOrmConfig(ormConfig *OrmConfig) {
	if ormConfig == nil {
		ormConfig = defOrmConfig()
	}
	if ormConfig.Pool == nil {
		ormConfig.Pool = defOrmPool()
	}
	app.setOrmConfig(ormConfig)
}

// 设置redis配置
func SetRedisConfig(redisConfig *RedisConfig) {
	if redisConfig == nil {
		redisConfig = defRedisConfig()
	}
	app.setRedisConfig(redisConfig)
}

// 设置跨域配置
func SetCorsConfig(corsConfig *CorsConfig) {
	if corsConfig == nil {
		corsConfig = defCorsConfig()
	}
	if len(corsConfig.AllowOrigin) == 0 {
		corsConfig.AllowOrigin = defCorsConfig().AllowOrigin
	}
	if len(corsConfig.AllowedMethods) == 0 {
		corsConfig.AllowedMethods = defCorsConfig().AllowedMethods
	}
	app.setCorsConfig(corsConfig)
}

// 设置httpclient配置
func SetCurlConfig(curlConfig *CurlConfig) {
	if curlConfig == nil {
		curlConfig = defCurlConfig()
	}
	app.setCurlConfig(curlConfig)
}

// 设置JWT配置
func SetJwtConfig(jwtConfig *JwtConfig) {
	if jwtConfig == nil {
		jwtConfig = defJwtConfig()
	}
	app.setJwtConfig(jwtConfig)
}

func GET(path string, handler Handler) {
	defRouterGroup.GET(path, handler)
}

func HEAD(path string, handler Handler) {
	defRouterGroup.HEAD(path, handler)
}

func POST(path string, handler Handler) {
	defRouterGroup.POST(path, handler)
}

func PUT(path string, handler Handler) {
	defRouterGroup.PUT(path, handler)
}

func PATCH(path string, handler Handler) {
	defRouterGroup.PATCH(path, handler)
}

func DELETE(path string, handler Handler) {
	defRouterGroup.DELETE(path, handler)
}

func ALL(path string, handler Handler) {
	defRouterGroup.ALL(path, handler)
}

// 启动服务
func Run() error {
	return app.run()
}

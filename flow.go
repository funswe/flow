package flow

import (
	"github.com/funswe/flow/log"
)

const (
	HttpMethodGet     = "GET"
	HttpMethodHead    = "HEAD"
	HttpMethodOptions = "OPTIONS"
	HttpMethodPost    = "POST"
	HttpMethodPut     = "PUT"
	HttpMethodPatch   = "PATCH"
	HttpMethodDelete  = "DELETE"
)

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
		orm:          defOrm(),
		redis:        defRedis(),
		curl:         defCurl(),
	}
)

func Use(m Middleware) {
	app.use(m)
}

func SetServerConfig(serverConfig *ServerConfig) {
	if serverConfig == nil {
		serverConfig = defServerConfig()
	}
	app.setServerConfig(serverConfig)
}

func SetLoggerConfig(loggerConfig *LoggerConfig) {
	if loggerConfig == nil {
		loggerConfig = defLoggerConfig()
	}
	app.setLoggerConfig(loggerConfig)
}

func SetOrmConfig(ormConfig *OrmConfig) {
	if ormConfig == nil {
		ormConfig = defOrmConfig()
	}
	if ormConfig.Pool == nil {
		ormConfig.Pool = defOrmPool()
	}
	app.setOrmConfig(ormConfig)
}

func SetRedisConfig(redisConfig *RedisConfig) {
	if redisConfig == nil {
		redisConfig = defRedisConfig()
	}
	app.setRedisConfig(redisConfig)
}

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

func SetCurlConfig(curlConfig *CurlConfig) {
	if curlConfig == nil {
		curlConfig = defCurlConfig()
	}
	app.setCurlConfig(curlConfig)
}

func Run() error {
	return app.run()
}

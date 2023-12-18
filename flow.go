package flow

import (
	"github.com/funswe/flow/log"
	"sync"
	"time"
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
	HttpHeaderXForwardedFor           = "X-Forwarded-For"
	HttpHeaderXRealIp                 = "X-Real-Ip"
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
		Orm:          defOrm(),
		Redis:        defRedis(),
		Curl:         defCurl(),
		Jwt:          defJwt(),
		beforeRuns:   make([]BeforeRun, 0),
	}
	defRouterGroup = &RouterGroup{}
	asyncTaskLock  = sync.RWMutex{}
	asyncTaskPool  = make(map[string]AsyncTask, 0)
	timerPool      = make(map[string]*timerJob, 0)
)

// Use 添加中间件
func Use(m Middleware) {
	defRouterGroup.Use(m)
}

// SetServerConfig 设置服务配置
func SetServerConfig(serverConfig *ServerConfig) {
	if serverConfig == nil {
		serverConfig = defServerConfig()
	}
	app.setServerConfig(serverConfig)
}

// SetLoggerConfig 设置日志配置
func SetLoggerConfig(loggerConfig *LoggerConfig) {
	if loggerConfig == nil {
		loggerConfig = defLoggerConfig()
	}
	app.setLoggerConfig(loggerConfig)
}

// SetOrmConfig 设置数据库配置
func SetOrmConfig(ormConfig *OrmConfig) {
	if ormConfig == nil {
		ormConfig = defOrmConfig()
	}
	if ormConfig.Pool == nil {
		ormConfig.Pool = defOrmPool()
	}
	app.setOrmConfig(ormConfig)
}

// SetRedisConfig 设置redis配置
func SetRedisConfig(redisConfig *RedisConfig) {
	if redisConfig == nil {
		redisConfig = defRedisConfig()
	}
	app.setRedisConfig(redisConfig)
}

// SetCorsConfig 设置跨域配置
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

// SetCurlConfig 设置httpclient配置
func SetCurlConfig(curlConfig *CurlConfig) {
	if curlConfig == nil {
		curlConfig = defCurlConfig()
	}
	app.setCurlConfig(curlConfig)
}

// SetJwtConfig 设置JWT配置
func SetJwtConfig(jwtConfig *JwtConfig) {
	if jwtConfig == nil {
		jwtConfig = defJwtConfig()
	}
	app.setJwtConfig(jwtConfig)
}

func GetApp() *Application {
	return app
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

// AddBefore 添加运行前需要执行的方法
func AddBefore(b BeforeRun) {
	app.addBefore(b)
}

// Run 启动服务
func Run() error {
	return app.run()
}

func ExecuteTask(task Task) {
	c := make(chan *TaskResult)
	go func() {
		if task.GetDelay() > 0 {
			<-time.After(task.GetDelay())
		}
		task.BeforeExecute(app)
		c <- task.Execute(app)
	}()
	go func() {
		select {
		case result := <-c:
			if task.IsTimeout() {
				return
			}
			task.AfterExecute(app)
			task.Completed(app, result)
		case <-time.After(task.GetTimeout()):
			task.Timeout(app)
		}
	}()
}

func ExecuteAsyncTask(task AsyncTask) {
	asyncTaskLock.Lock()
	if existTask, ok := asyncTaskPool[task.GetName()]; ok {
		existTask.Aggregation(app, task)
		asyncTaskLock.Unlock()
		return
	}
	asyncTaskPool[task.GetName()] = task
	asyncTaskLock.Unlock()
	c := make(chan *TaskResult)
	go func() {
		<-time.After(task.GetDelay())
		delete(asyncTaskPool, task.GetName())
		task.BeforeExecute(app)
		c <- task.Execute(app)
	}()
	go func() {
		select {
		case result := <-c:
			if task.IsTimeout() {
				return
			}
			task.AfterExecute(app)
			task.Completed(app, result)
		case <-time.After(task.GetTimeout()):
			task.Timeout(app)
		}
	}()
}

func StartTimer(timer Timer) {
	// 如果是周期的
	if timer.IsPeriodic() {
		// 如果是立即执行
		if timer.IsImmediately() {
			timer.Run(app)
		}
		ticker := time.NewTicker(timer.GetInterval())
		stopChan := make(chan bool, 0)
		tJob := &timerJob{
			stopChan: stopChan,
			timer:    timer,
		}
		timerPool[timer.GetName()] = tJob
		go func() {
			defer func() {
				ticker.Stop()
				delete(timerPool, tJob.timer.GetName())
				app.Logger.Infof("timer已停止，名称：%s", timer.GetName())
			}()
			for {
				select {
				case <-tJob.stopChan:
					return
				case <-ticker.C:
					tJob.timer.Run(app)
				}
			}
		}()
	} else {
		t := time.NewTimer(timer.GetInterval())
		go func() {
			<-t.C
			timer.Run(app)
		}()
	}
	app.Logger.Infof("timer已启动，名称：%s", timer.GetName())
}

func StopTimer(timerName string) {
	if len(timerName) == 0 {
		return
	}
	if v, ok := timerPool[timerName]; ok {
		v.stopChan <- true
	}
}

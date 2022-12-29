package flow

import "time"

type timerJob struct {
	stopChan chan bool
	timer    Timer
}

type Timer interface {
	// GetName 定时器的名称
	GetName() string
	// Run 执行定时器触发时的方法
	Run(app *Application)
	// GetInterval 获取定时器的周期
	GetInterval() time.Duration
	// IsPeriodic 是否是周期性的定时器
	IsPeriodic() bool
	// IsImmediately 是否立即执行
	IsImmediately() bool
}

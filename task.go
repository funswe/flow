package flow

import "time"

type TaskResult struct {
	Err  error
	Data map[string]interface{}
}

type Task interface {
	Start()
	Stop()
	Execute(app *Application) *TaskResult
	Completed(app *Application, result *TaskResult)
	Timeout(app *Application)
	GetTimeout() time.Duration
}

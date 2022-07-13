package flow

import "time"

type TaskResult struct {
	Err  error
	Data map[string]interface{}
}

type Task interface {
	Execute(app *Application, data map[string]interface{}) *TaskResult
	Completed(app *Application, result *TaskResult)
	Timeout(app *Application)
	GetTimeout() time.Duration
	GetExecuteData() map[string]interface{}
}

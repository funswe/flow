package flow

import "time"

type TaskResult struct {
	Err  error
	Data interface{}
}

type Task interface {
	BeforeExecute(app *Application)
	AfterExecute(app *Application)
	Execute(app *Application) *TaskResult
	Completed(app *Application, result *TaskResult)
	Timeout(app *Application)
	GetTimeout() time.Duration
	IsTimeout() bool
	GetDelay() time.Duration
}

type AsyncTask interface {
	GetName() string
	Aggregation(app *Application, newTask AsyncTask)
	BeforeExecute(app *Application)
	AfterExecute(app *Application)
	Execute(app *Application) *TaskResult
	Completed(app *Application, result *TaskResult)
	Timeout(app *Application)
	GetTimeout() time.Duration
	GetDelay() time.Duration
	IsTimeout() bool
}

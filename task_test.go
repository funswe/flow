package flow

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeTask struct {
	executed     chan struct{}
	completed    chan *TaskResult
	timedOut     chan struct{}
	beforeLeft   chan struct{}
	timeout      time.Duration
	delay        time.Duration
	executeDelay time.Duration
	isTimeout    bool
	result       *TaskResult
}

func newFakeTask() *fakeTask {
	return &fakeTask{
		executed:   make(chan struct{}, 1),
		completed:  make(chan *TaskResult, 1),
		timedOut:   make(chan struct{}, 1),
		beforeLeft: make(chan struct{}, 1),
		timeout:    time.Second,
		result:     &TaskResult{Data: "done"},
	}
}

func (f *fakeTask) BeforeExecute(app *Application) { f.beforeLeft <- struct{}{} }
func (f *fakeTask) AfterExecute(app *Application)  {}
func (f *fakeTask) Execute(app *Application) *TaskResult {
	f.executed <- struct{}{}
	if f.executeDelay > 0 {
		time.Sleep(f.executeDelay)
	}
	return f.result
}
func (f *fakeTask) Completed(app *Application, result *TaskResult) { f.completed <- result }
func (f *fakeTask) Timeout(app *Application)                       { f.timedOut <- struct{}{} }
func (f *fakeTask) GetTimeout() time.Duration                      { return f.timeout }
func (f *fakeTask) IsTimeout() bool                                { return f.isTimeout }
func (f *fakeTask) GetDelay() time.Duration                        { return f.delay }

func withNopLogger(t *testing.T) {
	t.Helper()
	original := app.Logger
	app.Logger = zap.NewNop()
	t.Cleanup(func() { app.Logger = original })
}

// 任务正常执行：BeforeExecute -> Execute -> Completed
func TestExecuteTaskCompleted(t *testing.T) {
	withNopLogger(t)
	task := newFakeTask()

	ExecuteTask(task)

	select {
	case <-task.beforeLeft:
	case <-time.After(2 * time.Second):
		t.Fatal("BeforeExecute未执行")
	}
	select {
	case <-task.executed:
	case <-time.After(2 * time.Second):
		t.Fatal("Execute未执行")
	}
	select {
	case result := <-task.completed:
		if result == nil || result.Data != "done" {
			t.Fatalf("任务结果不符：%+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Completed未执行")
	}
}

// 执行超时：Timeout 被调用，Completed 不再执行
func TestExecuteTaskTimeout(t *testing.T) {
	withNopLogger(t)
	task := newFakeTask()
	task.timeout = 20 * time.Millisecond
	task.executeDelay = 300 * time.Millisecond

	ExecuteTask(task)

	select {
	case <-task.timedOut:
	case <-time.After(2 * time.Second):
		t.Fatal("超时后应执行Timeout")
	}
	select {
	case <-task.completed:
		t.Fatal("超时后不应执行Completed")
	case <-time.After(100 * time.Millisecond):
	}
}

// 延迟任务：等待GetDelay之后才执行
func TestExecuteTaskWithDelay(t *testing.T) {
	withNopLogger(t)
	task := newFakeTask()
	task.delay = 50 * time.Millisecond

	begin := time.Now()
	ExecuteTask(task)

	select {
	case <-task.executed:
		if elapsed := time.Since(begin); elapsed < task.delay {
			t.Fatalf("延迟未生效，耗时 %v", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("延迟任务未执行")
	}
}

type fakeTimer struct {
	name      string
	interval  time.Duration
	periodic  bool
	immediate bool
	runCh     chan struct{}
}

func (f *fakeTimer) GetName() string            { return f.name }
func (f *fakeTimer) Run(app *Application)       { f.runCh <- struct{}{} }
func (f *fakeTimer) GetInterval() time.Duration { return f.interval }
func (f *fakeTimer) IsPeriodic() bool           { return f.periodic }
func (f *fakeTimer) IsImmediately() bool        { return f.immediate }

// 一次性定时器：到期执行一次
func TestStartTimerOnce(t *testing.T) {
	withNopLogger(t)
	timer := &fakeTimer{name: "unit-once", interval: 20 * time.Millisecond, runCh: make(chan struct{}, 4)}

	StartTimer(timer)

	select {
	case <-timer.runCh:
	case <-time.After(2 * time.Second):
		t.Fatal("一次性定时器未执行")
	}
}

// 周期定时器：可立即执行一次，并通过 StopTimer 停止
func TestStartTimerPeriodicAndStop(t *testing.T) {
	withNopLogger(t)
	timer := &fakeTimer{name: "unit-periodic", interval: 20 * time.Millisecond, periodic: true, immediate: true, runCh: make(chan struct{}, 64)}

	StartTimer(timer)

	// 立即执行的那一次
	select {
	case <-timer.runCh:
	case <-time.After(2 * time.Second):
		t.Fatal("周期定时器应立即执行一次")
	}
	// 周期触发
	select {
	case <-timer.runCh:
	case <-time.After(2 * time.Second):
		t.Fatal("周期定时器未按周期执行")
	}

	// 先把已产生的执行排空，再停止，避免缓冲里残留影响判定
	for len(timer.runCh) > 0 {
		<-timer.runCh
	}
	StopTimer(timer.name)

	time.Sleep(120 * time.Millisecond)
	if remain := len(timer.runCh); remain != 0 {
		t.Fatalf("定时器停止后仍在执行，剩余 %d 次", remain)
	}
}

// 停止不存在的定时器与空名称是安全操作
func TestStopTimerUnknown(t *testing.T) {
	StopTimer("not-exist")
	StopTimer("")
}

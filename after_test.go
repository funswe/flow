package flow

import "testing"

// 退出钩子按注册的逆序执行（后注册的先释放）
func TestAfterRunsExecuteInReverseOrder(t *testing.T) {
	original := app.afterRuns
	app.afterRuns = nil
	defer func() { app.afterRuns = original }()

	var order []string
	AddAfter(func(*Application) { order = append(order, "first") })
	AddAfter(func(*Application) { order = append(order, "second") })

	app.runAfterRuns()

	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Fatalf("退出钩子应按注册逆序执行，实际 %v", order)
	}
}

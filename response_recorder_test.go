package flow

import (
	"bytes"
	"net/http"
	"testing"
)

// responseWriterStub 记录底层writer收到的状态码与内容
type responseWriterStub struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseWriterStub() *responseWriterStub {
	return &responseWriterStub{header: make(http.Header)}
}

func (w *responseWriterStub) Header() http.Header            { return w.header }
func (w *responseWriterStub) WriteHeader(statusCode int)     { w.status = statusCode }
func (w *responseWriterStub) Write(data []byte) (int, error) { return w.body.Write(data) }

func TestResponseRecorderCapturesStatusAndBody(t *testing.T) {
	stub := newResponseWriterStub()
	recorder := NewResponseRecorder(stub, 0)

	recorder.WriteHeader(http.StatusForbidden)
	if _, err := recorder.Write([]byte("denied")); err != nil {
		t.Fatal(err)
	}

	if recorder.Status() != http.StatusForbidden {
		t.Fatalf("状态码应为403，实际 %d", recorder.Status())
	}
	if string(recorder.Body()) != "denied" {
		t.Fatalf("响应体应为denied，实际 %q", recorder.Body())
	}
	if stub.status != http.StatusForbidden || stub.body.String() != "denied" {
		t.Fatal("底层writer必须收到完整响应")
	}
}

func TestResponseRecorderImplicitStatusAndTruncate(t *testing.T) {
	stub := newResponseWriterStub()
	recorder := NewResponseRecorder(stub, 4)

	if _, err := recorder.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if recorder.Status() != http.StatusOK {
		t.Fatalf("未显式写状态码时应记为200，实际 %d", recorder.Status())
	}
	if string(recorder.Body()) != "abcd" {
		t.Fatalf("超过上限应只缓存前4字节，实际 %q", recorder.Body())
	}
	if !recorder.Truncated {
		t.Fatal("超过上限应置Truncated")
	}
	if stub.body.String() != "abcdef" {
		t.Fatal("截断只作用于缓存，底层writer仍应收到完整响应")
	}

	// 首个状态码为准，重复WriteHeader不覆盖
	recorder.WriteHeader(http.StatusInternalServerError)
	if recorder.Status() != http.StatusOK {
		t.Fatalf("重复WriteHeader不应覆盖首个状态码，实际 %d", recorder.Status())
	}
}

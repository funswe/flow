package json

import (
	"strings"
	"testing"
)

type sample struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Tags  []string `json:"tags,omitempty"`
	Skip  string   `json:"-"`
}

// Marshal/Unmarshal 与标准库语义兼容
func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	in := sample{Name: "flow", Count: 3, Tags: []string{"a", "b"}, Skip: "hidden"}
	raw, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal失败：%v", err)
	}
	if strings.Contains(string(raw), "hidden") {
		t.Fatalf("json:\"-\" 字段不应被序列化：%s", raw)
	}
	var out sample
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal失败：%v", err)
	}
	if out.Name != in.Name || out.Count != in.Count || len(out.Tags) != 2 {
		t.Fatalf("往返结果不一致：%+v", out)
	}
}

// 解析非法JSON应返回错误，而不是panic
func TestUnmarshalInvalidJSON(t *testing.T) {
	var out map[string]interface{}
	if err := Unmarshal([]byte("{invalid"), &out); err == nil {
		t.Fatal("非法JSON应返回错误")
	}
}

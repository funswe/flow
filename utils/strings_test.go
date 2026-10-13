package utils

import (
	"regexp"
	"testing"
)

// GetNanoid 生成非空、URL安全的唯一ID
func TestGetNanoid(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := GetNanoid()
		if len(id) == 0 {
			t.Fatal("nanoid不应为空")
		}
		if !pattern.MatchString(id) {
			t.Fatalf("nanoid应为URL安全字符，实际 %q", id)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("nanoid重复：%q", id)
		}
		seen[id] = struct{}{}
	}
}

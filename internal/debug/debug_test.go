//go:build !debug

package debug

import "testing"

func TestLogNoOp(t *testing.T) {
	// Should not panic or produce any side effects
	Log("test message")
	Log("formatted %s %d", "value", 42)
	Log("")
}

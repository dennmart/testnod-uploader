//go:build debug

package debug

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogWritesToStderr(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []any
		expected string
	}{
		{
			name:     "simple message",
			format:   "hello world",
			args:     nil,
			expected: "[DEBUG] hello world\n",
		},
		{
			name:     "formatted message",
			format:   "count: %d, name: %s",
			args:     []any{42, "test"},
			expected: "[DEBUG] count: 42, name: test\n",
		},
		{
			name:     "empty message",
			format:   "",
			args:     nil,
			expected: "[DEBUG] \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			origStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stderr = w

			Log(tt.format, tt.args...)

			w.Close()
			os.Stderr = origStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			got := buf.String()

			if !strings.Contains(got, tt.expected) {
				t.Errorf("Log() output = %q, expected to contain %q", got, tt.expected)
			}
		})
	}
}

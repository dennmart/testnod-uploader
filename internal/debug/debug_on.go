//go:build debug

package debug

import (
	"fmt"
	"os"
)

func Log(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}

package treasure

import (
	"os"
	"runtime"
	"testing"
)

func skipIfPermissionTestUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission tests do not apply on Windows or when running as root")
	}
}

package install

import (
	"os"
	"runtime"
	"testing"
)

func skipIfPermissionTestUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission tests are not reliable on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
}

func clearHomeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
}

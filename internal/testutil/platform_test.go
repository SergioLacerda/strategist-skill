package testutil

import "testing"

func TestSkipOnWindows_WindowsPathSkips(t *testing.T) {
	t.Run("windows", func(t *testing.T) {
		skipOnWindows(t, "windows", "expected test skip")
	})
}

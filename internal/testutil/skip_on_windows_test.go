package testutil

import "testing"

func TestSkipOnWindows_WindowsBranch(t *testing.T) {
	t.Run("simulated windows skip", func(t *testing.T) {
		skipOnWindows(t, "windows", "simulate skip on windows")
	})
}

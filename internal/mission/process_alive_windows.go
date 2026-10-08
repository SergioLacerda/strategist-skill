//go:build windows

package mission

import "golang.org/x/sys/windows"

const windowsStillActive = 259

func processAlive(pid int) bool {
	if pid <= 0 || uint64(pid) > uint64(^uint32(0)) {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid)) //nolint:gosec // PID range is checked above.
	if err != nil {
		return false
	}
	defer closeProcessHandle(handle)
	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == windowsStillActive
}

func closeProcessHandle(handle windows.Handle) {
	if err := windows.CloseHandle(handle); err != nil {
		// The process state has already been observed; cleanup failure cannot
		// change the answer and there is no useful recovery at this boundary.
		return
	}
}

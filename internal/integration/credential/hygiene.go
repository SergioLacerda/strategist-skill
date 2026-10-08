package credential

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Hygiene finding identifiers.
const (
	FindingLoosePermissions = "dotenv_loose_permissions"
	FindingNotIgnored       = "dotenv_not_git_ignored"
	FindingIgnoreUnknown    = "dotenv_ignore_state_unknown"
)

// Hygiene reports risks of a credential file without changing anything: loose
// permissions and a missing git ignore. A file that does not exist has none.
func Hygiene(path string, ignored func(string) (bool, error)) []string {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	var findings []string
	// Windows does not expose POSIX permission bits through FileMode; checking
	// them there would report every dotenv file as loose regardless of its ACL.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		findings = append(findings, FindingLoosePermissions)
	}
	isIgnored, err := ignored(path)
	switch {
	case err != nil:
		findings = append(findings, FindingIgnoreUnknown)
	case !isIgnored:
		findings = append(findings, FindingNotIgnored)
	}
	return findings
}

// GitIgnored asks git whether path is ignored. A directory outside a
// repository or a missing git is an error, which Hygiene reports as unknown.
func GitIgnored(path string) (bool, error) {
	cmd := exec.Command("git", "check-ignore", "-q", filepath.Base(path)) //nolint:gosec // G204: fixed command, path is the credential file name
	cmd.Dir = filepath.Dir(path)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git check-ignore: %w", err)
}

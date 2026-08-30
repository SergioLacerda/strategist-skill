package domain

// Compiler compiles all skill artifacts from a .strategist/ root directory.
type Compiler interface {
	CompileAll(root, indexPath string) error
}

// StaleChecker reports whether a compiled artifact is stale relative to its sources.
type StaleChecker interface {
	IsStale(artifactPath string) (bool, error)
}

// Installer installs the skill into a target directory.
type Installer interface {
	Install(cfg InstallConfig) error
}

// FileExtractor extracts embedded default files into a target directory.
// When force is false, files that already exist and differ from the embedded
// default are preserved (merge mode). When force is true, all files are overwritten.
type FileExtractor interface {
	Extract(targetDir string, force bool) error
	// ReadFile reads a single file from the embedded default FS without touching disk.
	// relPath is relative to the defaults root (e.g. "templates/epic-standalone.yaml").
	// Use this instead of reading from .strategist/ — that directory is write-only.
	ReadFile(relPath string) ([]byte, error)
}

// FileLister enumerates every regular file's path in the embedded default
// tree, relative to the defaults root. A separate, narrower interface from
// FileExtractor (rather than a third method on it) so existing
// FileExtractor test doubles that only ever needed Extract/ReadFile keep
// compiling unchanged — only `strategist upgrade`, which needs the full
// tree to detect orphaned files, requires a FileLister.
type FileLister interface {
	AllPaths() ([]string, error)
}

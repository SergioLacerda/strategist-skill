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
}

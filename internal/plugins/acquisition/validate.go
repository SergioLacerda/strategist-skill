package acquisition

import (
	"net/url"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const (
	// SourceGit identifies the first supported remote source class.
	SourceGit = "git"
)

const fullCommitBytes = 40

// Request is one explicit, immutable remote acquisition request.
type Request struct {
	Source         string
	Repository     string
	Commit         string
	PackageID      string
	AllowNetwork   bool
	QuarantineRoot string
	ForbiddenRoots []string
	Runner         GitRunner
}

// Result describes the reviewed quarantine transaction. It is not a trust
// decision and it does not imply promotion into external-skills-source.
type Result struct {
	QuarantinePath string
	Manifest       Manifest
}

func validateRequest(req Request) error {
	if req.Source != SourceGit {
		return acquisitionError("source_unsupported", "only source=git is supported")
	}
	if err := ValidateRepository(req.Repository); err != nil {
		return err
	}
	if err := ValidateCommit(req.Commit); err != nil {
		return err
	}
	if err := ValidatePackageID(req.PackageID); err != nil {
		return err
	}
	if !req.AllowNetwork {
		return acquisitionError("network_opt_in_required", "pass --allow-network explicitly")
	}
	return nil
}

// ValidateCommit accepts only a lowercase, full immutable Git object id.
func ValidateCommit(commit string) error {
	if len(commit) != fullCommitBytes {
		return acquisitionError("commit_invalid", "commit must be a full 40-character lowercase SHA-1")
	}
	for _, r := range commit {
		if !isLowerHex(r) {
			return acquisitionError("commit_invalid", "commit must be a full 40-character lowercase SHA-1")
		}
	}
	return nil
}

func isLowerHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
}

// ValidatePackageID uses the existing plugin path vocabulary and additionally
// rejects control characters and whitespace that would make operator paths
// ambiguous.
func ValidatePackageID(id string) error {
	if strings.TrimSpace(id) != id || id == "" {
		return acquisitionError("package_id_invalid", "package ID must be a non-empty relative path")
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return acquisitionError("package_id_invalid", "package ID must not contain whitespace or control characters")
		}
	}
	if err := domain.ValidatePluginRelativePath(id); err != nil {
		return acquisitionError("package_id_invalid", err.Error())
	}
	return nil
}

// ValidateRepository accepts supported remote Git URI forms and absolute
// local paths used only by hermetic tests. The commit, not the URI, provides
// the immutable content reference.
func ValidateRepository(repository string) error {
	if err := validateRepositoryText(repository); err != nil {
		return err
	}
	if isLocalRepository(repository) {
		return nil
	}
	return validateRepositoryURI(repository)
}

func validateRepositoryText(repository string) error {
	if strings.TrimSpace(repository) != repository || repository == "" || strings.HasPrefix(repository, "-") {
		return acquisitionError("repository_invalid", "repository must be a non-empty Git URI or absolute local path")
	}
	for _, r := range repository {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return acquisitionError("repository_invalid", "repository must not contain whitespace or control characters")
		}
	}
	return nil
}

func isLocalRepository(repository string) bool {
	return filepath.IsAbs(repository) || strings.HasPrefix(repository, "git@")
}

func validateRepositoryURI(repository string) error {
	parsed, err := url.Parse(repository)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "ssh") {
		return acquisitionError("repository_invalid", "repository must use https://, ssh://, git@ or an absolute local path")
	}
	if parsed.User != nil && parsed.User.Username() == "" {
		return acquisitionError("repository_invalid", "repository user is empty")
	}
	return nil
}

package credential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Env is the injectable view of the process a reference is resolved against.
// Nil members fall back to the operating system.
type Env struct {
	Getenv   func(string) string
	ReadFile func(string) ([]byte, error)
}

func (e Env) getenv(name string) string {
	if e.Getenv != nil {
		return e.Getenv(name)
	}
	return os.Getenv(name)
}

func (e Env) readFile(path string) ([]byte, error) {
	if e.ReadFile != nil {
		return e.ReadFile(path)
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path is an operator-configured credential reference
	if err != nil {
		return nil, fmt.Errorf("read credential file: %w", err)
	}
	return data, nil
}

func missing(reason string) error {
	return integration.NewError(integration.StateCredentialMissing, integration.Correlation{}, reason)
}

// Resolve reads a secret from an "env:NAME" or "dotenv:<path>#NAME" reference.
// An absent or empty value is credential_missing; nothing is exported.
func Resolve(ref string, env Env) (Secret, error) {
	scheme, rest, ok := strings.Cut(ref, ":")
	if !ok {
		return Secret{}, missing("credential reference is malformed")
	}
	switch scheme {
	case "env":
		return fromEnv(rest, env)
	case "dotenv":
		return fromDotenv(rest, env)
	default:
		return Secret{}, missing("credential reference scheme is not supported")
	}
}

func fromEnv(name string, env Env) (Secret, error) {
	if value := env.getenv(name); value != "" {
		return Secret{value: value}, nil
	}
	return Secret{}, missing("environment variable " + name + " is not set")
}

func fromDotenv(rest string, env Env) (Secret, error) {
	cut := strings.LastIndex(rest, "#")
	if cut <= 0 || cut == len(rest)-1 {
		return Secret{}, missing("dotenv reference needs <path>#<VARIABLE>")
	}
	path, name := rest[:cut], rest[cut+1:]
	data, err := env.readFile(path)
	if err != nil {
		return Secret{}, missing("dotenv file is not readable")
	}
	if value := ParseDotenv(string(data))[name]; value != "" {
		return Secret{value: value}, nil
	}
	return Secret{}, missing("variable " + name + " is not defined in the dotenv file")
}

// WorkspaceEnv resolves credential files relative to a workspace root, so
// "dotenv:.env#VAR" means the project's .env. The process environment is used
// as is for "env:" references.
func WorkspaceEnv(workspace string) Env {
	return Env{ReadFile: func(path string) ([]byte, error) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspace, path)
		}
		data, err := os.ReadFile(path) //nolint:gosec // G304: operator-configured credential reference
		if err != nil {
			return nil, fmt.Errorf("read credential file: %w", err)
		}
		return data, nil
	}}
}

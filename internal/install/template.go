package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// copyTemplate copies src (relative path inside strategistDir) to dst (filename inside strategistDir).
func copyTemplate(strategistDir, src, dst string) error {
	srcPath := filepath.Join(strategistDir, src)
	data, err := os.ReadFile(srcPath) //nolint:gosec // G304: path is constructed from install config, not user input
	if err != nil {
		return fmt.Errorf("read template %s: %w", src, err)
	}
	dstPath := filepath.Join(strategistDir, dst)
	if err := os.WriteFile(dstPath, data, 0o644); err != nil { //nolint:gosec // G703: path derived from trusted install config
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

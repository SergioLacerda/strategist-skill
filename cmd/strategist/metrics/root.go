package metrics

import (
	"fmt"
	"os"
)

func validateMetricsRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("root %q is not a directory", root)
	}
	return nil
}

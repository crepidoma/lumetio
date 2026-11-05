//go:build !windows

package localconf

import (
	"os"
	"path/filepath"
)

// documentsDir はドキュメントのフォルダを返す。
func documentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}

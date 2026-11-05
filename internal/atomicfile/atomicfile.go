// Package atomicfile は、ファイルを一時ファイルを通して書く。
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write は、同じフォルダの一時ファイルに書いてから置き換える。
// 書いている途中で止まっても、元のファイルが半端な中身にならないようにするため。
// フォルダがなければ作る。
func Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s を作れません: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("%s に書けません: %w", dir, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("%s に書けません: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s に書けません: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("%s に書けません: %w", path, err)
	}
	return nil
}

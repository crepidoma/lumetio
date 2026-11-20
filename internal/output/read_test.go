package output

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("bom.txt", bom+"本文\r\n所感\r\n")
	write("sjis.txt", "\x93\xfa\x95\xf1")
	// macOS で書いたときのように、濁点を分けて書く。
	write("nfd.txt", norm.NFD.String("デスト\n"))

	if text, _, err := ReadFile(dir, "nfd.txt"); err != nil || text != "デスト\n" {
		t.Errorf("nfd.txt = %q, %v", text, err)
	}
	// BOM を除き、改行は書かれたまま返す。
	if text, exists, err := ReadFile(dir, "bom.txt"); err != nil || !exists || text != "本文\r\n所感\r\n" {
		t.Errorf("bom.txt = %q, %v, %v", text, exists, err)
	}
	if text, exists, err := ReadFile(dir, "ない.txt"); err != nil || exists || text != "" {
		t.Errorf("ない.txt = %q, %v, %v", text, exists, err)
	}
	if _, exists, err := ReadFile(dir, "sjis.txt"); !exists || !errors.Is(err, ErrNotUTF8) {
		t.Errorf("sjis.txt: exists = %v, err = %v", exists, err)
	}
}

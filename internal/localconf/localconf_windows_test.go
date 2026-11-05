package localconf

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// 短い名前や大文字と小文字の違う書き方でも、同じフォルダなら同じとする。
// APPDATA が短い名前で渡されると、初期値と同じフォルダを選んでも違うフォルダに見えていた。
func TestSameDirWindowsNames(t *testing.T) {
	long := filepath.Join(t.TempDir(), "Long Folder Name")
	if err := os.Mkdir(long, 0o755); err != nil {
		t.Fatal(err)
	}
	short := shortName(t, long)
	if strings.EqualFold(short, long) {
		t.Skip("この環境では短い名前が作られない")
	}
	tests := [][2]string{
		{short, long},
		{strings.ToUpper(long), long},
		{filepath.Join(short, "data"), filepath.Join(long, "data")},
		{filepath.Join(long, "DATA"), filepath.Join(long, "data")},
	}
	for _, tt := range tests {
		if !SameDir(tt[0], tt[1]) {
			t.Errorf("SameDir(%q, %q) = false", tt[0], tt[1])
		}
	}
}

func shortName(t *testing.T, path string) string {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint16, syscall.MAX_PATH)
	n, err := syscall.GetShortPathName(p, &buf[0], uint32(len(buf)))
	if err != nil {
		t.Fatal(err)
	}
	return syscall.UTF16ToString(buf[:n])
}

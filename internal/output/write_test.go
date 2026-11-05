package output

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var outputTime = time.Date(2026, 11, 2, 15, 30, 12, 0, time.Local)

func results() []Result {
	return []Result{
		{FileName: "report_20260309.txt", Text: "日報\n", encoding: UTF8, newline: CRLF},
		{FileName: "effort_20260309.csv", Text: "a,b\n", encoding: UTF8BOM, newline: LF},
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestWrite(t *testing.T) {
	// 出力先のフォルダがなければ作る。
	dir := filepath.Join(t.TempDir(), "lumetio")
	written, err := Write(dir, results(), ConflictAsk, outputTime)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(written, []string{"report_20260309.txt", "effort_20260309.csv"}) {
		t.Errorf("written = %v", written)
	}
	if got := readFile(t, filepath.Join(dir, "report_20260309.txt")); got != "日報\r\n" {
		t.Errorf("report = %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "effort_20260309.csv")); got != "\xef\xbb\xbfa,b\n" {
		t.Errorf("effort = %q", got)
	}
}

// 1 つでもファイルがあれば、何も書かずに、あるファイルを返す。
func TestWriteAsk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "effort_20260309.csv"), []byte("前の"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Write(dir, results(), ConflictAsk, outputTime)
	var exists *ExistsError
	if !errors.As(err, &exists) || !slices.Equal(exists.Files, []string{"effort_20260309.csv"}) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "report_20260309.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("report が書かれている: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "effort_20260309.csv")); got != "前の" {
		t.Errorf("effort = %q", got)
	}
}

func TestWriteOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "effort_20260309.csv"), []byte("前の"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, results(), ConflictOverwrite, outputTime); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "effort_20260309.csv")); got != "\xef\xbb\xbfa,b\n" {
		t.Errorf("effort = %q", got)
	}
}

// 別名にするのは、すでにあるファイルだけ。
func TestWriteRename(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "effort_20260309.csv"), []byte("前の"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := Write(dir, results(), ConflictRename, outputTime)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(written, []string{"report_20260309.txt", "effort_20260309_20261102153012.csv"}) {
		t.Errorf("written = %v", written)
	}
	if got := readFile(t, filepath.Join(dir, "effort_20260309.csv")); got != "前の" {
		t.Errorf("前のファイルが変わった: %q", got)
	}
}

// 途中で失敗したら、書けたファイルを知らせる。
func TestWritePartialFailure(t *testing.T) {
	dir := t.TempDir()
	// 2 つ目と同じ名前のフォルダを置いて、書けなくする。
	if err := os.Mkdir(filepath.Join(dir, "effort_20260309.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	written, err := Write(dir, results(), ConflictOverwrite, outputTime)
	var we *WriteError
	if !errors.As(err, &we) {
		t.Fatalf("err = %v", err)
	}
	if !slices.Equal(written, []string{"report_20260309.txt"}) || !slices.Equal(we.Written, written) {
		t.Errorf("written = %v, WriteError.Written = %v", written, we.Written)
	}
}

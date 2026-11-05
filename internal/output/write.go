package output

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/atomicfile"
)

// Conflict は、出力先に同じ名前のファイルがあるときの扱い。
type Conflict int

const (
	// ConflictAsk は、何も書かずに ExistsError を返す。
	// 画面ならダイアログで、MCP なら AI がユーザーに尋ねてから、扱いを選んで呼び直す。
	ConflictAsk Conflict = iota
	ConflictOverwrite
	// ConflictRename は、拡張子の前に出力した時刻を付けた名前で書く。
	ConflictRename
)

// ParseConflict は、扱いの名前 (overwrite、rename、ask か空なら確かめる) を Conflict にする。
// 画面と MCP で、同じ名前を同じ扱いにするため。知らない名前なら ok は false。
func ParseConflict(s string) (c Conflict, ok bool) {
	switch s {
	case "", "ask":
		return ConflictAsk, true
	case "overwrite":
		return ConflictOverwrite, true
	case "rename":
		return ConflictRename, true
	}
	return ConflictAsk, false
}

// ExistsError は、出力先に同じ名前のファイルがあったことを表す。
type ExistsError struct {
	// Files は、すでにあるファイルの名前。
	Files []string
}

func (e *ExistsError) Error() string {
	return fmt.Sprintf("出力先に同じ名前のファイルがあります: %s", strings.Join(e.Files, "、"))
}

// WriteError は、書き込みの途中で失敗したことを表す。
type WriteError struct {
	// Written は、失敗する前に書けたファイルの名前。
	Written []string
	Err     error
}

func (e *WriteError) Error() string {
	if len(e.Written) == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("%v (書けたファイル: %s)", e.Err, strings.Join(e.Written, "、"))
}

func (e *WriteError) Unwrap() error {
	return e.Err
}

// Write は作った出力を dir に書き、書いたファイルの名前を返す。
// 出力はすべて作り終えてから渡す。1 つでも作れなかったら呼ばない。
// 同じファイル名になる出力がないことは、呼ぶ側が出力の名前と一緒に確かめておく。
// 同じ名前のファイルがあるときの確認は、出力セットで 1 回にまとめる。
// now は、別名に付ける出力した時刻。
func Write(dir string, results []Result, conflict Conflict, now time.Time) ([]string, error) {
	var exists []string
	for _, r := range results {
		_, err := os.Stat(filepath.Join(dir, r.FileName))
		if err == nil {
			exists = append(exists, r.FileName)
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s を確かめられません: %w", filepath.Join(dir, r.FileName), err)
		}
	}
	if len(exists) > 0 && conflict == ConflictAsk {
		return nil, &ExistsError{Files: exists}
	}

	var written []string
	for _, r := range results {
		name := r.FileName
		if conflict == ConflictRename && slices.Contains(exists, name) {
			name = renamed(name, now)
		}
		if err := atomicfile.Write(filepath.Join(dir, name), r.Bytes()); err != nil {
			return written, &WriteError{Written: written, Err: err}
		}
		written = append(written, name)
	}
	return written, nil
}

// renamed は、拡張子の前に _yyyyMMddHHmmss の形で時刻を付ける。
func renamed(name string, now time.Time) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + now.Format("_20060102150405") + ext
}

package gui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/store"
)

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 同じ名前のファイルがあれば書かずに知らせ、書いたら標準のアプリで開く。
func TestExport(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	write(t, root, "templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}}\n{{end}}")
	write(t, root, "days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 片付け\n")
	now := time.Date(2026, 11, 5, 15, 20, 0, 0, time.Local)
	var opened []string
	s := New(app.New(store.New(root), out, func() time.Time { return now }), nil, Env{Open: func(path string) error {
		opened = append(opened, path)
		return nil
	}})

	r, err := s.Export("2026-11-05", "日報テキスト", "ask")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Written, []string{"report_20261105.txt"}) || !slices.Equal(opened, []string{filepath.Join(out, "report_20261105.txt")}) {
		t.Errorf("1 回目: %+v, opened = %v", r, opened)
	}

	opened = nil
	r, err = s.Export("2026-11-05", "日報テキスト", "ask")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Exists, []string{"report_20261105.txt"}) || len(r.Written) != 0 || len(opened) != 0 {
		t.Errorf("2 回目: %+v, opened = %v", r, opened)
	}

	r, err = s.Export("2026-11-05", "日報テキスト", "rename")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Written, []string{"report_20261105_20261105152000.txt"}) || len(opened) != 1 {
		t.Errorf("別名: %+v, opened = %v", r, opened)
	}
}

// ペインの幅と右ペインを畳んでいるかは、起動したときの値を返し、保存すると local.yaml に書く。
func TestPanes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.yaml")
	s := New(nil, nil, Env{Local: localconf.NewFile(path), Panes: &localconf.Panes{Left: 300}})
	if got := s.Panes(); got != (Panes{Left: 300}) {
		t.Errorf("Panes = %+v", got)
	}
	if err := s.SavePanes(Panes{Left: 240, Right: 420, RightCollapsed: true}); err != nil {
		t.Fatal(err)
	}
	if got := s.Panes(); got != (Panes{Left: 240, Right: 420, RightCollapsed: true}) {
		t.Errorf("保存した後: %+v", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "panes:\n  left: 240\n  right: 420\n  right_collapsed: true\n" {
		t.Errorf("local.yaml = %q", data)
	}
}

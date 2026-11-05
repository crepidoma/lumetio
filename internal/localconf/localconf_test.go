package localconf

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func abs(parts ...string) string {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func TestParse(t *testing.T) {
	base := Local{DataDir: abs("既定", "data"), OutputDir: abs("既定", "out")}

	l := base
	data := "data_dir: " + abs("共有", "lumetio") + "\n"
	if err := parse([]byte(data), &l); err != nil {
		t.Fatal(err)
	}
	// 書かれていない項目は初期値のまま。
	want := Local{DataDir: abs("共有", "lumetio"), OutputDir: base.OutputDir}
	if l != want {
		t.Errorf("parse = %+v, want %+v", l, want)
	}

	l = base
	if err := parse(nil, &l); err != nil || l != base {
		t.Errorf("空のファイル: %+v, %v", l, err)
	}
}

func TestParseErrors(t *testing.T) {
	for _, data := range []string{
		"data_dir: lumetio\n",
		"output_dir: ./out\n",
		"window: 1\n",
	} {
		l := Local{}
		if err := parse([]byte(data), &l); err == nil {
			t.Errorf("parse(%q) succeeded, want error", data)
		}
	}
}

func TestLoadMissingCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.yaml")
	l, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if l != want {
		t.Errorf("Load = %+v, want %+v", l, want)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("local.yaml ができている: %v", err)
	}
}

func TestDefaults(t *testing.T) {
	l, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(l.DataDir) != "data" || filepath.Base(filepath.Dir(l.DataDir)) != "lumetio" {
		t.Errorf("DataDir = %s", l.DataDir)
	}
	if filepath.Base(l.OutputDir) != "lumetio" || !filepath.IsAbs(l.OutputDir) {
		t.Errorf("OutputDir = %s", l.OutputDir)
	}
}

func TestParseWindowAndPanes(t *testing.T) {
	l := Local{}
	data := "window:\n  width: 1000\n  height: 700\n  x: -10\n  y: 20\n  maximized: true\npanes:\n  left: 300\n  right_collapsed: true\n"
	if err := parse([]byte(data), &l); err != nil {
		t.Fatal(err)
	}
	if *l.Window != (Window{Width: 1000, Height: 700, X: -10, Y: 20, HasPosition: true, Maximized: true}) || *l.Panes != (Panes{Left: 300, RightCollapsed: true}) {
		t.Errorf("Window = %+v, Panes = %+v", *l.Window, *l.Panes)
	}

	// 位置は x と y がそろっているときだけ使う。
	l = Local{}
	if err := parse([]byte("window:\n  x: 10\n"), &l); err != nil {
		t.Fatal(err)
	}
	if l.Window.HasPosition {
		t.Errorf("Window = %+v", *l.Window)
	}
}

// アプリが書くときは、書かれていた場所の設定を残し、初期値を書き足さない。
func TestFileSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lumetio", "local.yaml")
	f := NewFile(path)

	if err := f.SavePanes(Panes{Left: 280, Right: 400}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveWindow(Window{Width: 1200, Height: 760, X: 30, Y: 40, HasPosition: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "window:\n  width: 1200\n  height: 760\n  x: 30\n  y: 40\n  maximized: false\npanes:\n  left: 280\n  right: 400\n  right_collapsed: false\n"
	if string(got) != want {
		t.Errorf("1 回目:\n%s\nwant:\n%s", got, want)
	}

	out := abs("出力")
	if err := os.WriteFile(path, []byte("output_dir: "+out+"\npanes:\n  left: 280\n  right: 400\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.SavePanes(Panes{Left: 220, Right: 300, RightCollapsed: true}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	l := Local{}
	if err := parse(got, &l); err != nil {
		t.Fatal(err)
	}
	if l.OutputDir != out || l.DataDir != "" || *l.Panes != (Panes{Left: 220, Right: 300, RightCollapsed: true}) || l.Window != nil {
		t.Errorf("2 回目: %s", got)
	}

	// 読めない local.yaml は書き換えない。
	if err := os.WriteFile(path, []byte("unknown: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.SavePanes(Panes{Left: 220, Right: 300}); err == nil {
		t.Error("読めない local.yaml を書き換えた")
	}
}

// 場所は、初期値と違うものだけを書き、ほかの項目は残す。
func TestSaveDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "local.yaml")
	f := NewFile(path)
	if err := f.SavePanes(Panes{Left: 280, Right: 400}); err != nil {
		t.Fatal(err)
	}
	def, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}

	data := abs("共有", "lumetio")
	if err := f.SaveDirs(data, def.OutputDir); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if l.DataDir != data || l.OutputDir != def.OutputDir || l.Panes == nil || l.Panes.Left != 280 {
		t.Errorf("1 回目: %s", raw)
	}
	if strings.Contains(string(raw), "output_dir") {
		t.Errorf("初期値の出力先を書いた: %s", raw)
	}

	// 初期値に戻すと、書いていた場所を消す。
	if err := f.SaveDirs(def.DataDir, def.OutputDir); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(path)
	if strings.Contains(string(raw), "data_dir") {
		t.Errorf("初期値に戻したのに残った: %s", raw)
	}

	if err := f.SaveDirs("lumetio", def.OutputDir); err == nil {
		t.Error("相対パスを書いた")
	}
}

func TestSameDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "データ")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	tests := []struct {
		a, b string
		want bool
	}{
		{dir, dir + string(filepath.Separator), true},
		{dir, filepath.Join(dir, "..", "データ"), true},
		{dir, other, false},
		// まだないフォルダも、親が同じなら同じ。
		{filepath.Join(dir, "ない"), filepath.Join(root, ".", "データ", "ない"), true},
		{filepath.Join(dir, "ない"), filepath.Join(other, "ない"), false},
		{"", "", true},
		{dir, "", false},
	}
	for _, tt := range tests {
		if got := SameDir(tt.a, tt.b); got != tt.want {
			t.Errorf("SameDir(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

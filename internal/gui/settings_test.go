package gui

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/store"
)

// 場所を変えると local.yaml に書き、以後の操作は新しいデータフォルダを読む。
func TestSaveLocations(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	write(t, first, "settings.yaml", "duration_format: \"m'm'\"\n")
	write(t, second, "settings.yaml", "duration_format: \"H'h'\"\n")
	path := filepath.Join(t.TempDir(), "local.yaml")
	out := filepath.Join(t.TempDir(), "out")
	s := New(app.New(store.New(first), out, nil), nil, Env{
		Local:     localconf.NewFile(path),
		Locations: Locations{DataDir: first, OutputDir: out},
	})

	if err := s.SaveLocations(Locations{DataDir: filepath.Join(first, "ない"), OutputDir: out}); !app.IsInputError(err) {
		t.Errorf("ないフォルダ: err = %v, want input error", err)
	}
	if got, _ := s.DurationFormat(); got != "m'm'" {
		t.Errorf("失敗した後に場所が変わった: %q", got)
	}

	want := Locations{DataDir: second, OutputDir: out}
	if err := s.SaveLocations(want); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.DurationFormat(); got != "H'h'" {
		t.Errorf("新しいデータフォルダを読んでいない: %q", got)
	}
	if got, _ := s.Locations(); got != want {
		t.Errorf("Locations = %+v", got)
	}
	l, err := localconf.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if l.DataDir != second || l.OutputDir != out {
		t.Errorf("local.yaml = %+v", l)
	}
}

// データフォルダの初期値は、まだなくても選べる。初めて起動したときと同じく、空のデータから始める。
func TestSaveLocationsDefaultDataDir(t *testing.T) {
	first := t.TempDir()
	def := filepath.Join(t.TempDir(), "data")
	out := filepath.Join(t.TempDir(), "out")
	s := New(app.New(store.New(first), out, nil), nil, Env{
		Local:            localconf.NewFile(filepath.Join(t.TempDir(), "local.yaml")),
		Locations:        Locations{DataDir: first, OutputDir: out},
		DefaultLocations: Locations{DataDir: def, OutputDir: out},
	})
	if err := s.SaveLocations(Locations{DataDir: def, OutputDir: out}); err != nil {
		t.Fatalf("初期値: %v", err)
	}
	// 書き方が違っても、初期値のフォルダなら初期値として扱う。
	written := filepath.Join(def, "..", "data")
	if err := s.SaveLocations(Locations{DataDir: written, OutputDir: out}); err != nil {
		t.Fatalf("書き方の違う初期値: %v", err)
	}
	if got := s.AtDefault(Locations{DataDir: written, OutputDir: first}); got != (LocationsAtDefault{DataDir: true}) {
		t.Errorf("AtDefault = %+v", got)
	}
	if err := s.SaveLocations(Locations{DataDir: filepath.Join(def, "ない"), OutputDir: out}); !app.IsInputError(err) {
		t.Errorf("初期値でないないフォルダ: err = %v, want input error", err)
	}
}

// ないフォルダから開き始めるとダイアログが開けないので、ある親から開く。
func TestChooseFolderStartsFromExistingDir(t *testing.T) {
	root := t.TempDir()
	var start string
	s := New(nil, nil, Env{ChooseFolder: func(_, current string) (string, error) {
		start = current
		return "", nil
	}})
	tests := []struct{ current, want string }{
		{root, root},
		{filepath.Join(root, "ない", "lumetio"), root},
		{"", ""},
	}
	for _, tt := range tests {
		if _, err := s.ChooseFolder("出力先", tt.current); err != nil {
			t.Fatal(err)
		}
		if start != tt.want {
			t.Errorf("%q から: 開き始め = %q, want %q", tt.current, start, tt.want)
		}
	}
}

func TestPreviewDurationFormat(t *testing.T) {
	s := New(nil, nil, Env{})
	got, err := s.PreviewDurationFormat("H'h'")
	if err != nil {
		t.Fatal(err)
	}
	want := []DurationSample{{90, "1.5h"}, {75, "1.25h"}, {50, "0.83h"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	// 空は、保存したときと同じく省略時の表記で書く。
	if got, _ := s.PreviewDurationFormat(""); got[0].Text != "1:30" {
		t.Errorf("空: %+v", got)
	}
	if _, err := s.PreviewDurationFormat("yyyy"); !app.IsInputError(err) {
		t.Errorf("読めない書式: err = %v", err)
	}
}

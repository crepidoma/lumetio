package gui

import (
	"reflect"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/store"
)

// 画面の行で読んだ定義を、そのまま書いて読み直せる。
func TestSaveNoteDefRows(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "settings.yaml", "notes:\n  - name: 在宅\n    type: bool\n    carry_over: true\n  - name: 連絡\n")
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{})

	base, err := s.NoteDefRows()
	if err != nil {
		t.Fatal(err)
	}
	want := []NoteDefRow{{Name: "在宅", Type: "bool", CarryOver: true}, {Name: "連絡", Type: "text"}}
	if !reflect.DeepEqual(base, want) {
		t.Fatalf("NoteDefRows = %+v", base)
	}
	next := []NoteDefRow{base[1], base[0], {Name: "ひとこと", Type: "text"}}
	if err := s.SaveNoteDefs(base, next); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.NoteDefRows(); !reflect.DeepEqual(got, next) {
		t.Errorf("書いた後 = %+v", got)
	}
}

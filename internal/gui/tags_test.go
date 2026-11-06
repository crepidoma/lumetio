package gui

import (
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/store"
)

// 画面で読んだ一覧を、そのまま書いて読み直せる。日を開いたときにも同じ一覧を返す。
func TestSaveTagRows(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "settings.yaml", "tags: [ふりかえり, 調査]\n")
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{})

	base, err := s.TagRows()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(base, []string{"ふりかえり", "調査"}) {
		t.Fatalf("TagRows = %q", base)
	}
	next := []string{"調査", "ふりかえり", "設計"}
	if err := s.SaveTags(base, next); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.TagRows(); !slices.Equal(got, next) {
		t.Errorf("書いた後 = %q", got)
	}
	o, err := s.Open("2026-11-04")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o.Tags, next) {
		t.Errorf("Open の Tags = %q", o.Tags)
	}
}

// 一覧がないときも、画面が null を受け取らないよう空の一覧を返す。
func TestTagRowsEmpty(t *testing.T) {
	s := New(app.New(store.New(t.TempDir()), t.TempDir(), nil), nil, Env{})
	got, err := s.TagRows()
	if err != nil || got == nil {
		t.Errorf("TagRows = %#v, %v", got, err)
	}
	o, err := s.Open("2026-11-04")
	if err != nil || o.Tags == nil {
		t.Errorf("Open の Tags = %#v, %v", o.Tags, err)
	}
}

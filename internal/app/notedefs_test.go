package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/settings"
)

func TestSaveNoteDefs(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\nnotes:\n  在宅: true\n  連絡: 遅れます\n")
	base, err := f.app.NoteDefs()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 2 || base[0].Name != "在宅" {
		t.Fatalf("NoteDefs = %+v", base)
	}

	// 名前を変え、並べ替え、欄を足す。ほかの項目と日次ファイルはそのまま残す。
	next := []settings.NoteDef{
		{Name: " 勤怠連絡 ", Type: settings.NoteText},
		base[0],
		{Name: "ひとこと", Type: settings.NoteText, CarryOver: true},
	}
	dayPath := filepath.Join(f.root, "days", "2026", "2026-11-04.yaml")
	day, err := os.ReadFile(dayPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.SaveNoteDefs(base, next); err != nil {
		t.Fatal(err)
	}
	st, _, err := f.store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	want := []settings.NoteDef{{Name: "勤怠連絡", Type: settings.NoteText}, base[0], next[2]}
	if !slices.Equal(st.Notes, want) || len(st.Projects) != 2 || st.DefaultOutput != "日報" {
		t.Errorf("settings = %+v", st)
	}
	if got, _ := os.ReadFile(dayPath); string(got) != string(day) {
		t.Errorf("日次ファイルが変わった:\n%s", got)
	}

	// 開いた後に他で定義が変わっていたら書かない。
	if err := f.app.SaveNoteDefs(base, base); !errors.Is(err, ErrConflict) {
		t.Errorf("他で変わった: err = %v, want ErrConflict", err)
	}

	for _, bad := range [][]settings.NoteDef{
		{{Name: " ", Type: settings.NoteText}},
		{{Name: "a", Type: settings.NoteText}, {Name: "a ", Type: settings.NoteBool}},
		{{Name: "a", Type: "choice"}},
	} {
		if err := f.app.SaveNoteDefs(want, bad); !IsInputError(err) {
			t.Errorf("%+v: err = %v, want input error", bad, err)
		}
	}
}

// NFC にそろえると同じになる名前は、重なりとして保存しない。
func TestSaveNoteDefsNFC(t *testing.T) {
	f := newFixture(t)
	base, err := f.app.NoteDefs()
	if err != nil {
		t.Fatal(err)
	}
	defs := []settings.NoteDef{{Name: "ガイド", Type: settings.NoteText}, {Name: "ガイド", Type: settings.NoteText}}
	if err := f.app.SaveNoteDefs(base, defs); !IsInputError(err) {
		t.Errorf("err = %v, want input error", err)
	}
}

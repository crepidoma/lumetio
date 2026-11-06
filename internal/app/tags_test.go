package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSaveTags(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 調査\n    tags: [設計]\n")
	base, err := f.app.Tags()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 0 {
		t.Fatalf("Tags = %q", base)
	}

	// 名前の前後の空白を除いて書く。ほかの項目と日次ファイルはそのまま残す。
	dayPath := filepath.Join(f.root, "days", "2026", "2026-11-04.yaml")
	day, err := os.ReadFile(dayPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.SaveTags(base, []string{" ふりかえり ", "調査"}); err != nil {
		t.Fatal(err)
	}
	st, _, err := f.store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ふりかえり", "調査"}
	if !slices.Equal(st.Tags, want) || len(st.Projects) != 2 || len(st.Notes) != 2 || st.DefaultOutput != "日報" {
		t.Errorf("settings = %+v", st)
	}
	if got, _ := os.ReadFile(dayPath); string(got) != string(day) {
		t.Errorf("日次ファイルが変わった:\n%s", got)
	}

	// タグを消しても、記録に付いているタグは残す。
	if err := f.app.SaveTags(want, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dayPath); string(got) != string(day) {
		t.Errorf("日次ファイルが変わった:\n%s", got)
	}

	// 開いた後に他で一覧が変わっていたら書かない。
	if err := f.app.SaveTags(want, want); !errors.Is(err, ErrConflict) {
		t.Errorf("他で変わった: err = %v, want ErrConflict", err)
	}

	for _, bad := range [][]string{
		{" "},
		{"a", "a "},
		// NFC にそろえると同じになる名前も重なりとする。
		{"ガイド", "ガイド"},
	} {
		if err := f.app.SaveTags(nil, bad); !IsInputError(err) {
			t.Errorf("%q: err = %v, want input error", bad, err)
		}
	}
}

// 日を開くと、タグの一覧も返す。
func TestOpenReturnsTags(t *testing.T) {
	f := newFixture(t)
	f.write("settings.yaml", settingsYAML+"tags: [ふりかえり, 調査]\n")
	o, err := f.app.Open(today)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o.Tags, []string{"ふりかえり", "調査"}) {
		t.Errorf("Tags = %q", o.Tags)
	}
}

package app

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
)

func titlesOf[T any](list []T, title func(T) string) []string {
	var out []string
	for _, v := range list {
		out = append(out, title(v))
	}
	return out
}

func TestReorderRecords(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n    time: ['09:00-09:15']\n  - title: 調査\n  - title: 資料作成\n")

	if err := f.app.ReorderRecords(today, []string{"資料作成", "朝会", "調査"}); err != nil {
		t.Fatal(err)
	}
	day := f.day(today)
	if got := titlesOf(day.Tasks, func(t dayfile.Task) string { return t.Title }); !slices.Equal(got, []string{"資料作成", "朝会", "調査"}) {
		t.Errorf("Tasks = %v", got)
	}
	// 中身は動かした記録についていく。
	if times(day.Tasks[1]) != "09:00-09:15" {
		t.Errorf("朝会 = %+v", day.Tasks[1])
	}

	// 画面に出していた記録とファイルの記録が食い違えば、他で変わったものとして書かない。
	for name, titles := range map[string][]string{
		"足りない":   {"朝会", "調査"},
		"知らない名前": {"朝会", "調査", "ない"},
		"重なり":    {"朝会", "朝会", "調査"},
	} {
		if err := f.app.ReorderRecords(today, titles); !errors.Is(err, ErrConflict) {
			t.Errorf("%s: err = %v, want conflict", name, err)
		}
	}
}

// 並びが変わらなければ書かない。記録のない日のファイルや、items.yaml を作らない。
func TestReorderUnchanged(t *testing.T) {
	f := newFixture(t)
	if err := f.app.ReorderRecords(today, nil); err != nil {
		t.Fatal(err)
	}
	if f.exists(today) {
		t.Error("記録のない日のファイルを作った")
	}
	if err := os.Remove(f.store.ItemsPath()); err != nil {
		t.Fatal(err)
	}
	if err := f.app.ReorderFavorites(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.store.ItemsPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("items.yaml: err = %v, want not exist", err)
	}
}

func TestReorderFavorites(t *testing.T) {
	f := newFixture(t)
	// よく使うタスクと、そうでないタスクが混ざっている。
	f.write("items.yaml", "- title: A\n  favorite: true\n- title: 朝会\n  time: ['09:00-09:15']\n  repeat:\n    freq: daily\n- title: B\n  favorite: true\n- title: C\n  favorite: true\n")

	if err := f.app.ReorderFavorites([]string{"C", "A", "B"}); err != nil {
		t.Fatal(err)
	}
	got := titlesOf(f.items(), func(it items.Item) string { return it.Title })
	// よく使うタスクが入っていた位置に、新しい順で入れ直す。朝会の位置は変わらない。
	if !slices.Equal(got, []string{"C", "朝会", "A", "B"}) {
		t.Errorf("items = %v", got)
	}
	if err := f.app.ReorderFavorites([]string{"A", "B"}); !errors.Is(err, ErrConflict) {
		t.Errorf("足りない: err = %v, want conflict", err)
	}
}

package app

import (
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
)

func (f *fixture) items() []items.Item {
	f.t.Helper()
	list, _, err := f.store.LoadItems()
	if err != nil {
		f.t.Fatal(err)
	}
	return list
}

func findItem(list []items.Item, title string) (items.Item, bool) {
	i := slices.IndexFunc(list, func(it items.Item) bool { return it.Title == title })
	if i < 0 {
		return items.Item{}, false
	}
	return list[i], true
}

func TestSetFavorite(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 調査\n    project:\n      code: OLD-01\n    time: [30m]\n")
	// その日の記録がいちばん新しい。
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 調査\n    project:\n      code: DEV-001\n      name: 新機能\n    tags: [ふりかえり]\n    exclude: true\n    note: 写さない\n    time: [30m]\n")

	// まだ items.yaml にないタスクは、いちばん新しい記録から写す。
	if _, err := f.app.SetFavorite(today, "調査", true); err != nil {
		t.Fatal(err)
	}
	it, ok := findItem(f.items(), "調査")
	if !ok || !it.Favorite || it.Project != (dayfile.Project{Code: "DEV-001", Name: "新機能"}) || !slices.Equal(it.Tags, []string{"ふりかえり"}) || !it.Exclude || len(it.Time) != 0 {
		t.Errorf("調査 = %+v", it)
	}

	// プロジェクトなどは設定ではないので、外して設定がなくなれば、写した値ごと消える。
	if _, err := f.app.SetFavorite(today, "調査", false); err != nil {
		t.Fatal(err)
	}
	if it, ok := findItem(f.items(), "調査"); ok {
		t.Errorf("調査が items.yaml に残っている: %+v", it)
	}

	// 記録のないタスクも入れられ、外すと設定が残らないので消える。
	if _, err := f.app.SetFavorite(today, "新しいタスク", true); err != nil {
		t.Fatal(err)
	}
	if it, ok := findItem(f.items(), "新しいタスク"); !ok || !it.Favorite || it.Project.Code != "" {
		t.Errorf("新しいタスク = %+v, %v", it, ok)
	}
	if _, err := f.app.SetFavorite(today, "新しいタスク", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := findItem(f.items(), "新しいタスク"); ok {
		t.Error("新しいタスクが items.yaml に残っている")
	}

	// items.yaml にないタスクを外しても、何も書かない。
	before := f.items()
	if _, err := f.app.SetFavorite(today, "ない", false); err != nil {
		t.Fatal(err)
	}
	if len(f.items()) != len(before) {
		t.Error("items.yaml が変わった")
	}
	if _, err := f.app.SetFavorite(today, " ", true); !IsInputError(err) {
		t.Errorf("空の名前: err = %v", err)
	}
}

func TestCandidates(t *testing.T) {
	f := newFixture(t)
	f.write("days/2025/2025-03-01.yaml", "date: 2025-03-01\ntasks:\n  - title: 古い調査\n  - title: 朝会\n")
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 朝会\n    project:\n      code: MTG\n  - title: 資料作成\n")
	f.write("days/2026/2026-11-03.yaml", "date: 2026-11-03\ntasks: [")

	list, skipped, err := f.app.Candidates()
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, c := range list {
		titles = append(titles, c.Title)
	}
	// 新しい日から、items.yaml だけにあるものは後ろ。
	want := []string{"朝会", "資料作成", "古い調査", "週次ミーティング", "問い合わせ対応"}
	if !slices.Equal(titles, want) {
		t.Errorf("titles = %v, want %v", titles, want)
	}
	if list[0].Project.Code != "MTG" || !list[0].LastDate.Equal(today.AddDate(0, 0, -1)) || !list[3].LastDate.IsZero() {
		t.Errorf("list = %+v", list)
	}
	// 記録にもある items.yaml のタスクも、items.yaml にあるものとして返す。タスクの編集を開けるかを決めるため。
	var inItems []string
	for _, c := range list {
		if c.InItems {
			inItems = append(inItems, c.Title)
		}
	}
	if want := []string{"朝会", "週次ミーティング", "問い合わせ対応"}; !slices.Equal(inItems, want) {
		t.Errorf("InItems = %v, want %v", inItems, want)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %v", skipped)
	}
}

// 左ペインで印を付けるため、その日にもう記録があるタスクを返す。
func TestTasksRecorded(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 前の日\n")
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n  - title: 調査\n")
	lists, err := f.app.Tasks(today)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(lists.Recorded, []string{"朝会", "調査"}) {
		t.Errorf("Recorded = %v", lists.Recorded)
	}
}

package app

import (
	"slices"
	"strings"
	"testing"
)

// 時間不定は untimed: true で明示する。時間も時間不定もない記録は時間未入力 (時間の入れ忘れ)。

const untimedItems = `- title: 情報収集
  untimed: true
  favorite: true
- title: 問い合わせ対応
  time: [30m]
  favorite: true
- title: 朝会
  time: ['09:00-09:15']
  favorite: true
`

// record は、その日の記録を名前で探す。
func (f *fixture) record(title string) (untimed bool, times int) {
	f.t.Helper()
	for _, t := range f.day(today).Tasks {
		if t.Title == title {
			return t.Untimed, len(t.Time)
		}
	}
	f.t.Fatalf("%s の記録がない", title)
	return false, 0
}

// 画面の＋は足した後に時間を打つので、時間がなければ時間未入力で足す。
// いつもの時間を時間不定にしたタスクは、時間不定で入る。
func TestAddRecordsUntimed(t *testing.T) {
	f := newFixture(t)
	f.write("items.yaml", untimedItems)
	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "調査"}, {Title: "情報収集"}}); err != nil {
		t.Fatal(err)
	}
	if u, n := f.record("調査"); u || n != 0 {
		t.Errorf("調査 = %v/%d, want 時間未入力", u, n)
	}
	if u, n := f.record("情報収集"); !u || n != 0 {
		t.Errorf("情報収集 = %v/%d, want 時間不定", u, n)
	}
}

// MCP から足すときは、時間か時間不定がはっきり決まらない記録を作らない。
// AI が時間を入れ忘れた記録が、黙って時間未入力や時間不定にならないようにするため。
func TestAddRecordsExplicit(t *testing.T) {
	f := newFixture(t)
	f.write("items.yaml", untimedItems)

	errs := map[string][]NewRecord{
		"時間も時間不定もない":      {{Title: "調査"}},
		"空の時間":            {{Title: "調査", Time: ptr("")}},
		"時間と時間不定の両方":      {{Title: "調査", Time: ptr("30m"), Untimed: ptr(true)}},
		"1 つでも誤りがあれば書かない": {{Title: "設計", Time: ptr("30m")}, {Title: "調査"}},
	}
	for name, recs := range errs {
		if _, err := f.app.AddRecordsExplicit(today, recs); !IsInputError(err) {
			t.Errorf("%s: err = %v, want 入力の誤り", name, err)
		}
	}
	if f.exists(today) {
		t.Fatal("誤りのときにファイルを作った")
	}

	if _, err := f.app.AddRecordsExplicit(today, []NewRecord{
		{Title: "調査", Untimed: ptr(true)},
		{Title: "情報収集"},
		{Title: "問い合わせ対応"},
		// いつもの時間があっても、時間不定を指定すれば時間不定で入る。
		{Title: "朝会", Untimed: ptr(true)},
	}); err != nil {
		t.Fatal(err)
	}
	for title, want := range map[string]bool{"調査": true, "情報収集": true, "問い合わせ対応": false, "朝会": true} {
		u, n := f.record(title)
		if u != want || (want && n != 0) || (!want && n == 0) {
			t.Errorf("%s = %v/%d, want 時間不定 %v", title, u, n, want)
		}
	}

	// もう記録があるタスクに、足す時間がなければ誤り。
	if _, err := f.app.AddRecordsExplicit(today, []NewRecord{{Title: "調査"}}); !IsInputError(err) {
		t.Errorf("足す時間がない: err = %v", err)
	}
}

// 時間不定の記録に時間を足すと、時間を決めたことになるので、時間不定は外れる。
// 時間がある記録は、時間と時間不定の両方を持たないよう、時間不定にできない。
func TestAddRecordsUntimedToExisting(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 調査\n    untimed: true\n  - title: 設計\n  - title: 実装\n    time: [30m]\n")

	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "調査", Time: ptr("45m")}, {Title: "設計", Untimed: ptr(true)}}); err != nil {
		t.Fatal(err)
	}
	if u, n := f.record("調査"); u || n != 1 {
		t.Errorf("調査 = %v/%d, want 時間あり", u, n)
	}
	if u, n := f.record("設計"); !u || n != 0 {
		t.Errorf("設計 = %v/%d, want 時間不定", u, n)
	}
	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "実装", Untimed: ptr(true)}}); !IsInputError(err) {
		t.Errorf("時間がある記録を時間不定に: err = %v", err)
	}
}

func TestUpdateRecordUntimed(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 調査\n  - title: 実装\n    time: [30m]\n  - title: 設計\n    untimed: true\n")

	if _, err := f.app.UpdateRecord(today, "調査", RecordPatch{Untimed: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if u, _ := f.record("調査"); !u {
		t.Error("調査を時間不定にできない")
	}
	if _, err := f.app.UpdateRecord(today, "実装", RecordPatch{Untimed: ptr(true)}); !IsInputError(err) {
		t.Errorf("時間がある記録を時間不定に: err = %v", err)
	}
	// 同じ指定で時間を消せば、時間不定にできる。
	if _, err := f.app.UpdateRecord(today, "実装", RecordPatch{Time: ptr(""), Untimed: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if u, n := f.record("実装"); !u || n != 0 {
		t.Errorf("実装 = %v/%d, want 時間不定", u, n)
	}
	if _, err := f.app.UpdateRecord(today, "設計", RecordPatch{Time: ptr("1h")}); err != nil {
		t.Fatal(err)
	}
	if u, n := f.record("設計"); u || n != 1 {
		t.Errorf("設計 = %v/%d, want 時間を入れたら時間不定は外れる", u, n)
	}
	if _, err := f.app.UpdateRecord(today, "調査", RecordPatch{Untimed: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if u, _ := f.record("調査"); u {
		t.Error("調査の時間不定を外せない")
	}
}

// 計上する記録の時間未入力は、アプリの外へ出す前に気づけるよう、日の警告と出力の警告に入れる。
// 時間不定と、計上しない記録は、工数の取り込みに関わらないので警告しない。
func TestMissingTimeWarnings(t *testing.T) {
	f := newFixture(t)
	f.write("templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}}\n{{end}}")
	f.write("templates/日報.set.yaml", "kind: set\noutputs: [日報テキスト]\n")
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 調査\n  - title: 設計\n    untimed: true\n  - title: 雑談\n    exclude: true\n  - title: 実装\n    time: [30m]\n")

	want := []string{"調査 の時間が入っていません"}
	view, err := f.app.Day(today)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(view.Warnings, want) {
		t.Errorf("Day の警告 = %q", view.Warnings)
	}
	r, err := f.app.Render(today, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Warnings, want) {
		t.Errorf("Render の警告 = %q", r.Warnings)
	}
	// 記録を書いた直後には出さない。画面の＋は時間未入力で足すので、足すたびに出ると邪魔になるため。
	warnings, err := f.app.AddRecords(today, []NewRecord{{Title: "レビュー"}})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(warnings, func(w string) bool { return strings.Contains(w, "時間が入っていません") }) {
		t.Errorf("AddRecords の警告 = %q", warnings)
	}
}

// いつもの時間を時間不定にすることは、それだけでタスクの設定になる。いつもの時間とは一緒に持たない。
func TestSaveTaskUntimed(t *testing.T) {
	f := newFixture(t)
	if err := f.app.SaveTask("", nil, TaskInput{Title: "情報収集", Untimed: true}); err != nil {
		t.Fatal(err)
	}
	it, err := f.app.Item("情報収集")
	if err != nil {
		t.Fatal(err)
	}
	if !it.Untimed || len(it.Time) != 0 {
		t.Errorf("情報収集 = %+v", it)
	}
	if err := f.app.SaveTask("", nil, TaskInput{Title: "設計", Time: "30m", Untimed: true}); !IsInputError(err) {
		t.Errorf("いつもの時間と時間不定の両方: err = %v", err)
	}
}

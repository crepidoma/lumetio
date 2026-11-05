package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/output"
	"github.com/crepidoma/lumetio/internal/store"
)

// 2026-11-05 は木曜日。今は 15:20。
var (
	today = time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	now   = time.Date(2026, 11, 5, 15, 20, 0, 0, time.Local)
)

const settingsYAML = `projects:
  - code: DEV-001
    name: 新機能
  - code: OLD-01
    name: 終わった案件
    until: 2026-10-31
notes:
  - name: 在宅
    type: bool
    carry_over: true
  - name: 連絡
default_output: 日報
`

const itemsYAML = `- title: 朝会
  project:
    code: MTG
    name: 会議
  time: ['09:00-09:15']
  repeat:
    freq: weekly
    by_day: [mo, tu, we, th, fr]
- title: 週次ミーティング
  time: ['13:00-14:00']
  repeat:
    freq: weekly
    by_day: [th]
- title: 問い合わせ対応
  project:
    code: SUP
  time: [30m]
  favorite: true
`

type fixture struct {
	t     *testing.T
	root  string
	out   string
	app   *App
	store *store.Store
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{t: t, root: root, out: filepath.Join(t.TempDir(), "out")}
	f.write("settings.yaml", settingsYAML)
	f.write("items.yaml", itemsYAML)
	f.store = store.New(root)
	f.app = New(f.store, f.out, func() time.Time { return now })
	return f
}

func (f *fixture) write(name, content string) {
	f.t.Helper()
	path := filepath.Join(f.root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) day(date time.Time) dayfile.Day {
	f.t.Helper()
	day, v, err := f.store.LoadDay(date)
	if err != nil {
		f.t.Fatal(err)
	}
	if !v.Exists() {
		f.t.Fatalf("%s の日次ファイルがない", date.Format("2006-01-02"))
	}
	return day
}

func (f *fixture) exists(date time.Time) bool {
	_, err := os.Stat(f.store.DayPath(date))
	return err == nil
}

func ptr[T any](v T) *T {
	return &v
}

func task(day dayfile.Day, title string) dayfile.Task {
	for _, t := range day.Tasks {
		if t.Title == title {
			return t
		}
	}
	return dayfile.Task{}
}

func times(t dayfile.Task) string {
	var ss []string
	for _, e := range t.Time {
		ss = append(ss, e.String())
	}
	return strings.Join(ss, ",")
}

// 日を開いただけではファイルを作らない。予定と、作るときに書くメモ欄の値を返す。
func TestDayWithoutFile(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\nnotes:\n  在宅: true\n  連絡: 明日は休み\n")

	view, err := f.app.Day(today)
	if err != nil {
		t.Fatal(err)
	}
	if view.Exists || f.exists(today) {
		t.Error("ファイルができている")
	}
	var plans []string
	for _, p := range view.Plans {
		plans = append(plans, p.Title)
	}
	if !slices.Equal(plans, []string{"朝会", "週次ミーティング"}) {
		t.Errorf("Plans = %v", plans)
	}
	want := []dayfile.Note{
		{Name: "在宅", Kind: dayfile.NoteBool, Bool: true},
		{Name: "連絡", Kind: dayfile.NoteText},
	}
	if !slices.Equal(view.Notes, want) {
		t.Errorf("Notes = %+v", view.Notes)
	}
}

func TestAddRecords(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\nnotes:\n  在宅: true\n")
	// 14 日より前の記録からも、プロジェクトなどを写す。
	f.write("days/2026/2026-09-01.yaml", "date: 2026-09-01\ntasks:\n  - title: 調査\n    project:\n      code: DEV-001\n      name: 新機能\n    tags: [ふりかえり]\n    note: 写さない\n    time: [60m]\n")

	_, err := f.app.AddRecords(today, []NewRecord{
		// 予定と同じタスクは、いつもの時間で記録になる。
		{Title: "朝会"},
		// 時間を指定すれば、その時間で記録になる。
		{Title: "週次ミーティング", Time: ptr("13:00-14:30")},
		{Title: "調査", Time: ptr("-12:00")},
		{Title: "新しいタスク", Time: ptr("14:30-"), Project: &dayfile.Project{Code: "DEV-001"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	day := f.day(today)

	if got := task(day, "朝会"); got.Project.Code != "MTG" || times(got) != "09:00-09:15" {
		t.Errorf("朝会 = %+v", got)
	}
	if got := task(day, "週次ミーティング"); times(got) != "13:00-14:30" {
		t.Errorf("週次ミーティング = %+v", got)
	}
	// -12:00 は、直前の記録 (朝会) の終わりから。
	if got := task(day, "調査"); got.Project.Code != "DEV-001" || !slices.Equal(got.Tags, []string{"ふりかえり"}) || got.Note != "" || times(got) != "09:15-12:00" {
		t.Errorf("調査 = %+v", got)
	}
	// 14:30- は今 (15:20) まで。名前はプロジェクトの一覧から入る。
	if got := task(day, "新しいタスク"); got.Project != (dayfile.Project{Code: "DEV-001", Name: "新機能"}) || times(got) != "14:30-15:20" {
		t.Errorf("新しいタスク = %+v", got)
	}

	// ファイルを作るときに、メモ欄を前回から引き継いで書く。空の欄は書かない。
	if !slices.Equal(day.Notes, []dayfile.Note{{Name: "在宅", Kind: dayfile.NoteBool, Bool: true}}) {
		t.Errorf("Notes = %+v", day.Notes)
	}
	view, err := f.app.Day(today)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Plans) != 0 {
		t.Errorf("Plans = %+v", view.Plans)
	}
}

// 同じタスクの記録があれば、新しい行を作らず時間を足す。
// items.yaml にないタスクを足すときは、プロジェクトなどだけを前の記録から写し、時間とメモは写さない。その日ごとのものなので。
func TestAddRecordsDoesNotCopyTimeOrNote(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-03.yaml", "date: 2026-11-03\ntasks:\n  - title: 調査\n    project:\n      code: DEV-001\n    note: 写さない\n    time: [60m]\n")
	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "調査"}}); err != nil {
		t.Fatal(err)
	}
	if got := task(f.day(today), "調査"); got.Project.Code != "DEV-001" || len(got.Time) != 0 || got.Note != "" {
		t.Errorf("調査 = %+v", got)
	}
}

func TestAddRecordsToExisting(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 問い合わせ対応\n    project:\n      code: SUP\n    time: [30m]\n")

	if _, err := f.app.AddRecords(today, []NewRecord{
		{Title: "問い合わせ対応", Time: ptr("15")},
		// 時間を省略すると、いつもの時間を足す。
		{Title: "問い合わせ対応", Project: &dayfile.Project{Code: "SUP"}},
	}); err != nil {
		t.Fatal(err)
	}
	day := f.day(today)
	if len(day.Tasks) != 1 || times(day.Tasks[0]) != "30m,15m,30m" {
		t.Errorf("Tasks = %+v", day.Tasks)
	}
}

// 1 つでも誤りがあれば、何も書かない。
func TestAddRecordsAllOrNothing(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 問い合わせ対応\n    project:\n      code: SUP\n")
	before := f.day(today)

	for name, recs := range map[string][]NewRecord{
		"時間の書き方":       {{Title: "a", Time: ptr("30m")}, {Title: "b", Time: ptr("あとで")}},
		"タスク名がない":      {{Title: "a"}, {Title: " "}},
		"違うプロジェクト":     {{Title: "a"}, {Title: "問い合わせ対応", Project: &dayfile.Project{Code: "DEV-001"}}},
		"コードのないプロジェクト": {{Title: "a", Project: &dayfile.Project{Name: "名前だけ"}}},
		"記録がない":        {},
	} {
		_, err := f.app.AddRecords(today, recs)
		if !IsInputError(err) {
			t.Errorf("%s: err = %v, want input error", name, err)
		}
	}
	if after := f.day(today); !slices.EqualFunc(before.Tasks, after.Tasks, func(a, b dayfile.Task) bool { return a.Title == b.Title }) {
		t.Errorf("書き換わっている: %+v", after.Tasks)
	}
}

func TestUpdateRecord(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n    time: ['09:00-09:15']\n  - title: 調査\n    time: ['09:15-12:00']\n")

	if _, err := f.app.UpdateRecord(today, "調査", RecordPatch{
		// 自分の今の時間は「直前の記録」に含めないので、朝会の終わりから。
		Time:    ptr("-11:00, 30m"),
		Project: &dayfile.Project{Code: "DEV-001"},
		Tags:    &[]string{"ふりかえり", " ", "ふりかえり"},
		Note:    ptr("メモ"),
		Exclude: ptr(true),
	}); err != nil {
		t.Fatal(err)
	}
	got := task(f.day(today), "調査")
	if times(got) != "09:15-11:00,30m" || got.Project.Name != "新機能" || !slices.Equal(got.Tags, []string{"ふりかえり"}) || got.Note != "メモ" || !got.Exclude {
		t.Errorf("調査 = %+v", got)
	}

	// 空にすると時間不定、プロジェクトのコードを空にすると外す。
	if _, err := f.app.UpdateRecord(today, "調査", RecordPatch{Time: ptr(""), Project: &dayfile.Project{}}); err != nil {
		t.Fatal(err)
	}
	if got := task(f.day(today), "調査"); len(got.Time) != 0 || got.Project.Code != "" {
		t.Errorf("調査 = %+v", got)
	}

	if _, err := f.app.UpdateRecord(today, "ない", RecordPatch{Note: ptr("x")}); !IsInputError(err) {
		t.Errorf("ない記録: err = %v", err)
	}
	if _, err := f.app.UpdateRecord(today.AddDate(0, 0, 1), "調査", RecordPatch{Note: ptr("x")}); !IsInputError(err) {
		t.Errorf("ファイルのない日: err = %v", err)
	}
	if f.exists(today.AddDate(0, 0, 1)) {
		t.Error("ファイルのない日にファイルができた")
	}
}

// その日の記録の名前だけを直せる。タスクの設定の名前は変えない。
func TestUpdateRecordTitle(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 問い合せ対応\n    time: [30m]\n  - title: 調査\n")
	itemsBefore, err := os.ReadFile(filepath.Join(f.root, "items.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.app.UpdateRecord(today, "問い合せ対応", RecordPatch{Title: ptr(" 問い合わせ対応 ")}); err != nil {
		t.Fatal(err)
	}
	if got := task(f.day(today), "問い合わせ対応"); times(got) != "30m" {
		t.Errorf("問い合わせ対応 = %+v", got)
	}
	itemsAfter, err := os.ReadFile(filepath.Join(f.root, "items.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(itemsAfter) != string(itemsBefore) {
		t.Error("items.yaml が変わった")
	}

	for name, title := range map[string]string{"同じ名前": "調査", "空": " "} {
		if _, err := f.app.UpdateRecord(today, "問い合わせ対応", RecordPatch{Title: ptr(title)}); !IsInputError(err) {
			t.Errorf("%s: err = %v, want input error", name, err)
		}
	}
	// 名前を変えずに渡しても、誤りにしない。
	if _, err := f.app.UpdateRecord(today, "調査", RecordPatch{Title: ptr("調査")}); err != nil {
		t.Errorf("同じ名前のまま: %v", err)
	}
}

// 時間のチップを 1 つだけ直す。
func TestReplaceTime(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n    time: ['09:00-09:15']\n  - title: 調査\n    time: ['09:15-10:00', 30m, '13:00-14:00']\n")

	// 直す時間そのものは「直前の記録」に入れず、同じ記録のほかの時間は入れる。
	if _, err := f.app.ReplaceTime(today, "調査", 2, "13:00-14:00", "-15:00"); err != nil {
		t.Fatal(err)
	}
	if got := times(task(f.day(today), "調査")); got != "09:15-10:00,30m,10:00-15:00" {
		t.Errorf("調査 = %s", got)
	}
	// 複数書けば並びに入る。
	if _, err := f.app.ReplaceTime(today, "調査", 1, "30m", "15m 20m"); err != nil {
		t.Fatal(err)
	}
	if got := times(task(f.day(today), "調査")); got != "09:15-10:00,15m,20m,10:00-15:00" {
		t.Errorf("調査 = %s", got)
	}
	// 空なら消す。
	if _, err := f.app.ReplaceTime(today, "調査", 0, "09:15-10:00", ""); err != nil {
		t.Fatal(err)
	}
	if got := times(task(f.day(today), "調査")); got != "15m,20m,10:00-15:00" {
		t.Errorf("調査 = %s", got)
	}

	if _, err := f.app.ReplaceTime(today, "調査", 0, "15m", "あとで"); !IsInputError(err) {
		t.Errorf("書き方の誤り: err = %v, want input error", err)
	}

	// 画面で見ていた時間と、ファイルのその位置の時間が違えば、他で変わったものとして書かない。
	// 画面が古いまま他で時間が足されたときに、別の時間を消さないため。
	for name, tt := range map[string]struct {
		index   int
		current string
	}{
		"位置がずれた":  {0, "20m"},
		"位置が範囲の外": {3, "30m"},
		"負の位置":    {-1, "30m"},
	} {
		if _, err := f.app.ReplaceTime(today, "調査", tt.index, tt.current, ""); !errors.Is(err, ErrConflict) {
			t.Errorf("%s: err = %v, want ErrConflict", name, err)
		}
	}
	if got := times(task(f.day(today), "調査")); got != "15m,20m,10:00-15:00" {
		t.Errorf("断った後 = %s", got)
	}
}

// 入力中の解釈は、保存するときと同じ決まりで、ファイルを書かない。
func TestPreviewTime(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n    time: ['09:00-09:15']\n  - title: 調査\n    time: ['09:15-10:00', '13:00-14:00']\n")
	before := f.day(today)

	tests := []struct {
		title string
		index int
		input string
		want  string
	}{
		// 足すときは、すべての時間が「直前の記録」の候補。
		{"新しいタスク", -1, "-15:00", "14:00-15:00"},
		{"調査", -1, "-15:00", "14:00-15:00"},
		// 置き換えるときは、置き換える時間を候補に入れない。
		{"調査", 1, "-15:00", "10:00-15:00"},
		{"調査", 1, "", ""},
	}
	for _, tt := range tests {
		es, err := f.app.PreviewTime(today, tt.title, tt.index, tt.input)
		if err != nil {
			t.Errorf("%s %d %q: %v", tt.title, tt.index, tt.input, err)
			continue
		}
		var ss []string
		for _, e := range es {
			ss = append(ss, e.String())
		}
		if got := strings.Join(ss, ","); got != tt.want {
			t.Errorf("%s %d %q = %s, want %s", tt.title, tt.index, tt.input, got, tt.want)
		}
	}
	if _, err := f.app.PreviewTime(today, "調査", 2, "30m"); !IsInputError(err) {
		t.Errorf("位置が範囲の外: err = %v", err)
	}
	if _, err := f.app.PreviewTime(today, "調査", -1, "あとで"); !IsInputError(err) {
		t.Errorf("書き方の誤り: err = %v", err)
	}
	if after := f.day(today); times(after.Tasks[1]) != times(before.Tasks[1]) {
		t.Error("ファイルが書き換わった")
	}
}

// 期間の外のプロジェクト (期限切れ、開始前) のコードを、記録の行に印を付けられるように返す。
func TestDayOutOfPeriod(t *testing.T) {
	f := newFixture(t)
	f.write("settings.yaml", "projects:\n  - code: DEV-001\n  - code: OLD-01\n    until: 2026-10-31\n  - code: NEXT-01\n    start: 2026-11-06\n")
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: a\n    project:\n      code: OLD-01\n  - title: b\n    project:\n      code: DEV-001\n  - title: c\n    project:\n      code: OLD-01\n  - title: d\n    project:\n      code: NEXT-01\n")
	view, err := f.app.Day(today)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"プロジェクト OLD-01 の期間 (〜2026-10-31) の外です", "プロジェクト NEXT-01 の期間 (2026-11-06〜) の外です"}
	if !slices.Equal(view.OutOfPeriod, []string{"OLD-01", "NEXT-01"}) || !slices.Equal(view.Warnings, want) {
		t.Errorf("OutOfPeriod = %v, Warnings = %q", view.OutOfPeriod, view.Warnings)
	}
}

// 完了の印は、そのタスクを最後に記録した日にだけ付けられる。
func TestUpdateRecordDone(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-03.yaml", "date: 2026-11-03\ntasks:\n  - title: 調査\n")
	f.write("days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 調査\n")

	_, err := f.app.UpdateRecord(today.AddDate(0, 0, -2), "調査", RecordPatch{Done: ptr(true)})
	if !IsInputError(err) || !strings.Contains(err.Error(), "2026-11-04") {
		t.Errorf("err = %v", err)
	}
	if _, err := f.app.UpdateRecord(today.AddDate(0, 0, -1), "調査", RecordPatch{Done: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if !task(f.day(today.AddDate(0, 0, -1)), "調査").Done {
		t.Error("完了になっていない")
	}
	lists, err := f.app.Tasks(today)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists.Recent) != 0 {
		t.Errorf("Recent = %+v", lists.Recent)
	}
	// 外すこともできる。
	if _, err := f.app.UpdateRecord(today.AddDate(0, 0, -1), "調査", RecordPatch{Done: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if task(f.day(today.AddDate(0, 0, -1)), "調査").Done {
		t.Error("完了が外れていない")
	}
}

func TestRemoveRecord(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 朝会\n  - title: 調査\n")
	if err := f.app.RemoveRecord(today, "朝会"); err != nil {
		t.Fatal(err)
	}
	if day := f.day(today); len(day.Tasks) != 1 || day.Tasks[0].Title != "調査" {
		t.Errorf("Tasks = %+v", day.Tasks)
	}
	if err := f.app.RemoveRecord(today, "朝会"); !IsInputError(err) {
		t.Errorf("err = %v", err)
	}
}

func TestSkipPlans(t *testing.T) {
	f := newFixture(t)
	if err := f.app.SkipPlans(today, []string{"朝会", "問い合わせ対応"}); !IsInputError(err) {
		t.Errorf("予定でないもの: err = %v", err)
	}
	if f.exists(today) {
		t.Error("誤りがあったのにファイルができた")
	}
	// スキップだけでもファイルを作る。
	if err := f.app.SkipPlans(today, []string{"朝会", "週次ミーティング"}); err != nil {
		t.Fatal(err)
	}
	// 前回の日がないので、引き継ぐ値もなく、メモ欄は書かない。
	if day := f.day(today); !slices.Equal(day.Skipped, []string{"朝会", "週次ミーティング"}) || len(day.Notes) != 0 {
		t.Errorf("day = %+v", day)
	}
}

// 画面で見ていた値と、ファイルの値が違えば書かない。打っているあいだに他の PC が同じ欄を書き換えた値を、上書きしないため。
func TestSetNotesCurrent(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\nnotes:\n  連絡: 他の PC\n")
	shown := ""
	if err := f.app.SetNotes(today, []NoteValue{{Name: "連絡", Value: "こちら", Current: &shown}}); !errors.Is(err, ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
	shown = "他の PC"
	if err := f.app.SetNotes(today, []NoteValue{{Name: "連絡", Value: "こちら", Current: &shown}}); err != nil {
		t.Fatal(err)
	}
	if got := f.day(today).Notes; len(got) != 1 || got[0].Text != "こちら" {
		t.Errorf("Notes = %+v", got)
	}
}

func TestSetNotes(t *testing.T) {
	f := newFixture(t)
	for name, values := range map[string][]NoteValue{
		"定義にない欄": {{Name: "在宅勤務", Value: true}},
		"型が違う":   {{Name: "在宅", Value: "はい"}},
		"文字列でない": {{Name: "連絡", Value: true}},
		"欄がない":   {},
	} {
		if err := f.app.SetNotes(today, values); !IsInputError(err) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if f.exists(today) {
		t.Error("誤りがあったのにファイルができた")
	}
	if err := f.app.SetNotes(today, []NoteValue{{Name: "在宅", Value: true}, {Name: "連絡", Value: "なし"}}); err != nil {
		t.Fatal(err)
	}
	want := []dayfile.Note{{Name: "在宅", Kind: dayfile.NoteBool, Bool: true}, {Name: "連絡", Kind: dayfile.NoteText, Text: "なし"}}
	if got := f.day(today).Notes; !slices.Equal(got, want) {
		t.Errorf("Notes = %+v", got)
	}

	// 欄を空にしたら、その欄をファイルから消す。
	if err := f.app.SetNotes(today, []NoteValue{{Name: "在宅", Value: false}, {Name: "連絡", Value: ""}}); err != nil {
		t.Fatal(err)
	}
	if got := f.day(today).Notes; len(got) != 0 {
		t.Errorf("空にした後 = %+v", got)
	}
}

func TestProjects(t *testing.T) {
	f := newFixture(t)
	codes := func(date time.Time) []string {
		ps, err := f.app.Projects(date)
		if err != nil {
			t.Fatal(err)
		}
		var cs []string
		for _, p := range ps {
			cs = append(cs, p.Code)
		}
		return cs
	}
	// 期限は今日ではなく、その日と比べる。
	if got := codes(today); !slices.Equal(got, []string{"DEV-001"}) {
		t.Errorf("今日: %v", got)
	}
	if got := codes(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)); !slices.Equal(got, []string{"DEV-001", "OLD-01"}) {
		t.Errorf("期限の日: %v", got)
	}

	// 開始日より前の日は候補に出さず、開始日からは出す。
	f.write("settings.yaml", "projects:\n  - code: NEXT-01\n    start: 2026-11-06\n    until: 2026-11-30\n")
	if got := codes(today); len(got) != 0 {
		t.Errorf("開始前: %v", got)
	}
	if got := codes(time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)); !slices.Equal(got, []string{"NEXT-01"}) {
		t.Errorf("開始日: %v", got)
	}
}

func TestRenderAndExport(t *testing.T) {
	f := newFixture(t)
	f.write("templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}}\n{{end}}")
	f.write("templates/日報.set.yaml", "kind: set\noutputs: [日報テキスト]\n")
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 片付け\n    project:\n      code: OLD-01\n")

	// 名前を省略すると既定の出力。期間の外のプロジェクトは警告して、出力は止めない。
	r, err := f.app.Render(today, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Outputs) != 1 || r.Outputs[0].Name != "日報テキスト" || r.Outputs[0].Text != "片付け\n" {
		t.Errorf("Outputs = %+v", r.Outputs)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "OLD-01") {
		t.Errorf("Warnings = %v", r.Warnings)
	}
	if _, err := f.app.Render(today, "ない"); !IsInputError(err) {
		t.Errorf("ない名前: err = %v", err)
	}

	written, _, err := f.app.Export(today, "", output.ConflictAsk)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(written, []string{"report_20261105.txt"}) {
		t.Errorf("written = %v", written)
	}
	if _, _, err := f.app.Export(today, "", output.ConflictAsk); !errors.As(err, new(*output.ExistsError)) {
		t.Errorf("2 回目: err = %v", err)
	}
	written, _, err = f.app.Export(today, "日報", output.ConflictRename)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(written, []string{"report_20261105_20261105152000.txt"}) {
		t.Errorf("written = %v", written)
	}
}

func TestSkipPlansTwice(t *testing.T) {
	f := newFixture(t)
	for range 2 {
		if err := f.app.SkipPlans(today, []string{"朝会"}); err != nil {
			t.Fatal(err)
		}
	}
	if day := f.day(today); !slices.Equal(day.Skipped, []string{"朝会"}) {
		t.Errorf("Skipped = %v", day.Skipped)
	}
}

// 同じプロセスから同時に書いても、書き込みが失われない。
func TestConcurrentWrites(t *testing.T) {
	f := newFixture(t)
	const n = 20
	errs := make(chan error, n)
	for i := range n {
		go func() {
			_, err := f.app.AddRecords(today, []NewRecord{{Title: fmt.Sprintf("タスク%02d", i), Time: ptr("10m")}})
			errs <- err
		}()
	}
	for range n {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
	if got := len(f.day(today).Tasks); got != n {
		t.Errorf("記録が %d 件、want %d", got, n)
	}
}

// すでにある記録に時間を足すとき、プロジェクトはコードで比べる。
// 一覧で名前を変えただけで、同じプロジェクトを違うものとして断らないため。
func TestAddRecordsSameProjectRenamed(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 調査\n    project:\n      code: DEV-001\n      name: 前の名前\n    time: [30m]\n")
	code := dayfile.Project{Code: "DEV-001"}
	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "調査", Time: ptr("15m"), Project: &code}}); err != nil {
		t.Fatalf("名前を省いた: %v", err)
	}
	other := dayfile.Project{Code: "DEV-001", Name: "別の名前"}
	if _, err := f.app.AddRecords(today, []NewRecord{{Title: "調査", Time: ptr("15m"), Project: &other}}); !IsInputError(err) {
		t.Errorf("違う名前を指定した: err = %v, want input error", err)
	}
	if got := task(f.day(today), "調査"); got.Project.Name != "前の名前" || len(got.Time) != 2 {
		t.Errorf("調査 = %+v", got)
	}
}

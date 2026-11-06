package dayfile

import (
	"slices"
	"strings"
	"testing"

	"github.com/crepidoma/lumetio/internal/yamlfile"
)

const sample = `date: 2026-10-29
tasks:
  - title: 見積もりの作成
    project:
      code: DEV-001
      name: '[開発]サンプル案件'
    time: ['09:30-11:30']
  - title: 資料の整理
    project:
      code: X-001
      name: 事務作業
    time: [30m, 60m, '16:00-17:00']
  - title: 情報収集
    project:
      code: RESEARCH
    tags: [ふりかえり]
    note: |
      試作して、使えそうなことを確かめた。
      続きは明日。
  - title: ランチ会
    exclude: true
    done: true
    time: ['12:00-13:00']
  - title: 時間を決めない作業
    untimed: true
  - title: 時間を入れ忘れた作業
skipped: [定例会議]
notes:
  在宅: true
  ひとこと: 進んだ
`

func TestRoundTrip(t *testing.T) {
	day, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != sample {
		t.Errorf("Marshal(Parse(sample)) =\n%s\nwant\n%s", out, sample)
	}
}

func TestParse(t *testing.T) {
	day, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if got := day.Date.Format(yamlfile.DateLayout); got != "2026-10-29" {
		t.Errorf("Date = %s", got)
	}
	if len(day.Tasks) != 6 {
		t.Fatalf("len(Tasks) = %d", len(day.Tasks))
	}
	second := day.Tasks[1]
	if second.Project != (Project{Code: "X-001", Name: "事務作業"}) || len(second.Time) != 3 || second.Time[2].Minutes != 60 {
		t.Errorf("Tasks[1] = %+v", second)
	}
	if got := day.Tasks[2].Project.DisplayName(); got != "RESEARCH" {
		t.Errorf("名前のないプロジェクトの DisplayName = %q, want コード", got)
	}
	if day.Tasks[3].Project != (Project{}) {
		t.Errorf("Tasks[3].Project = %+v, want プロジェクトなし", day.Tasks[3].Project)
	}
	if !day.Tasks[3].Exclude || !day.Tasks[3].Done {
		t.Errorf("Tasks[3] の Exclude と Done = %v, %v, want true, true", day.Tasks[3].Exclude, day.Tasks[3].Done)
	}
	if day.Tasks[0].Done {
		t.Error("Tasks[0].Done = true, want 省略時 false")
	}
	if t4 := day.Tasks[4]; len(t4.Time) != 0 || !t4.Untimed {
		t.Errorf("Tasks[4] = %+v, want 時間不定", t4)
	}
	// 時間も時間不定もない記録は、時間未入力。
	if t5 := day.Tasks[5]; len(t5.Time) != 0 || t5.Untimed {
		t.Errorf("Tasks[5] = %+v, want 時間未入力", t5)
	}
	if day.Tasks[0].Untimed {
		t.Error("Tasks[0].Untimed = true, want 省略時 false")
	}
	if day.Tasks[2].Note != "試作して、使えそうなことを確かめた。\n続きは明日。\n" {
		t.Errorf("Tasks[2].Note = %q", day.Tasks[2].Note)
	}
	wantNotes := []Note{
		{Name: "在宅", Kind: NoteBool, Bool: true},
		{Name: "ひとこと", Kind: NoteText, Text: "進んだ"},
	}
	if len(day.Notes) != len(wantNotes) {
		t.Fatalf("Notes = %+v", day.Notes)
	}
	for i, w := range wantNotes {
		if day.Notes[i] != w {
			t.Errorf("Notes[%d] = %+v, want %+v", i, day.Notes[i], w)
		}
	}
}

// メモ欄の値の形は、日次ファイルの書き方から決まる。
func TestParseNoteKinds(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\nnotes:\n  b: True\n  c: 'true'\n  d: 12\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Note{
		{Name: "b", Kind: NoteBool, Bool: true},
		// 引用符で囲んだものと数字は、文字列として読む。
		{Name: "c", Kind: NoteText, Text: "true"},
		{Name: "d", Kind: NoteText, Text: "12"},
	}
	for i, w := range want {
		if day.Notes[i] != w {
			t.Errorf("Notes[%d] = %+v, want %+v", i, day.Notes[i], w)
		}
	}

	// 文字列の "true" は、書き出しても文字列のまま残る。
	out, err := Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if again.Notes[2] != want[2] || again.Notes[1] != want[1] {
		t.Errorf("書き出して読み直した Notes = %+v\n%s", again.Notes, out)
	}
}

// 手で書いたファイルは、時刻を引用符で囲まなくても読める。
func TestParseUnquotedTimes(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\ntasks:\n  - title: a\n    time: [09:30-11:30, 30m]\nnotes:\n  ひとこと: ''\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := day.Tasks[0].Time; len(got) != 2 || got[0].Minutes != 120 || got[1].Minutes != 30 {
		t.Errorf("Time = %+v", got)
	}
}

func TestParseNormalizesNFC(t *testing.T) {
	// 「が」を、macOS で出ることのある「か」と濁点の 2 文字で書く。
	decomposed := "date: 2026-10-29\ntasks:\n  - title: がいぎ\n"
	day, err := Parse([]byte(decomposed))
	if err != nil {
		t.Fatal(err)
	}
	if day.Tasks[0].Title != "がいぎ" {
		t.Errorf("Title = %q, want NFC", day.Tasks[0].Title)
	}
}

func TestParseOnlyDate(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "date: 2026-10-29\n" {
		t.Errorf("Marshal = %q", out)
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"知らないキー":           "date: 2026-10-29\nmemo: x\n",
		"タスクの知らないキー":       "date: 2026-10-29\ntasks:\n  - title: a\n    labels: [x]\n",
		"title がない":        "date: 2026-10-29\ntasks:\n  - project:\n      code: a\n",
		"プロジェクトに code がない": "date: 2026-10-29\ntasks:\n  - title: a\n    project:\n      name: b\n",
		"プロジェクトが文字列":       "date: 2026-10-29\ntasks:\n  - title: a\n    project: b\n",
		"プロジェクトの知らないキー":    "date: 2026-10-29\ntasks:\n  - title: a\n    project:\n      code: b\n      id: c\n",
		"タスク名が重なる":         "date: 2026-10-29\ntasks:\n  - title: a\n  - title: a\n",
		"日付が読めない":          "date: 2026/10/29\n",
		"日付がない":            "tasks: []\n",
		"時間が保存形式でない":       "date: 2026-10-29\ntasks:\n  - title: a\n    time: [9-10]\n",
		"時間と時間不定の両方":       "date: 2026-10-29\ntasks:\n  - title: a\n    time: [30m]\n    untimed: true\n",
		"notes が組でない":      "date: 2026-10-29\nnotes: [a]\n",
		"notes の値が入れ子":     "date: 2026-10-29\nnotes:\n  a: [x]\n",
		"notes の欄が重なる":     "date: 2026-10-29\nnotes:\n  連絡: a\n  連絡: b\n",
		"YAML として壊れている":    "date: [\n",
	}
	for name, input := range tests {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("%s: Parse succeeded, want error", name)
		}
	}
}

func TestMarshalQuotesAmbiguousText(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\ntasks:\n  - title: 'true'\n  - title: '123'\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse(Marshal) error: %v\n%s", err, out)
	}
	if again.Tasks[0].Title != "true" || again.Tasks[1].Title != "123" {
		t.Errorf("titles = %q, %q\n%s", again.Tasks[0].Title, again.Tasks[1].Title, out)
	}
	if !strings.Contains(string(out), `"true"`) {
		t.Errorf("true が文字列として書かれていない\n%s", out)
	}
}

// 空の欄は書いていない欄と同じ。手で書いた空の値は読まず、書くときも書かない。
func TestEmptyNotes(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\nnotes:\n  在宅: false\n  連絡: ''\n  ひとこと: 進んだ\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []Note{{Name: "ひとこと", Kind: NoteText, Text: "進んだ"}}; !slices.Equal(day.Notes, want) {
		t.Errorf("Notes = %+v", day.Notes)
	}

	day.Notes = []Note{{Name: "在宅", Kind: NoteBool}, {Name: "連絡", Kind: NoteText}}
	data, err := Marshal(day)
	if err != nil {
		t.Fatal(err)
	}
	if want := "date: 2026-10-29\n"; string(data) != want {
		t.Errorf("Marshal =\n%s", data)
	}

	// 空の値と同じ名前の欄が重なっていても、重なりとして知らせる。
	if _, err := Parse([]byte("date: 2026-10-29\nnotes:\n  連絡: ''\n  連絡: a\n")); err == nil {
		t.Error("重なりを読めた")
	}
}

// 時間と時間不定の両方を持つ記録は、書いても読めないファイルになるので、書く前に止める。
func TestMarshalRejectsTimeAndUntimed(t *testing.T) {
	day, err := Parse([]byte("date: 2026-10-29\ntasks:\n  - title: a\n    time: [30m]\n"))
	if err != nil {
		t.Fatal(err)
	}
	day.Tasks[0].Untimed = true
	if _, err := Marshal(day); err == nil {
		t.Error("Marshal succeeded, want error")
	}
}

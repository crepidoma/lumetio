package settings

import (
	"testing"
	"time"
)

const sample = `projects:
  - code: DEV-001
    name: '[開発]サンプルアプリ'
    start: 2026-04-01
    until: 2026-12-31
  - code: ADMIN
notes:
  - name: 在宅
    type: bool
    carry_over: true
  - name: 連絡
duration_format: H'h'm'm'
default_output: 日報
`

func TestRoundTrip(t *testing.T) {
	s, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != sample {
		t.Errorf("Marshal(Parse(sample)) =\n%s\nwant\n%s", out, sample)
	}
}

func TestParse(t *testing.T) {
	s, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	want := []Project{
		{Code: "DEV-001", Name: "[開発]サンプルアプリ", Start: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)},
		{Code: "ADMIN"},
	}
	if len(s.Projects) != len(want) {
		t.Fatalf("Projects = %+v", s.Projects)
	}
	for i, w := range want {
		got := s.Projects[i]
		if got.Code != w.Code || got.Name != w.Name || !got.Start.Equal(w.Start) || !got.Until.Equal(w.Until) {
			t.Errorf("Projects[%d] = %+v, want %+v", i, got, w)
		}
	}

	wantNotes := []NoteDef{
		{Name: "在宅", Type: NoteBool, CarryOver: true},
		{Name: "連絡", Type: NoteText},
	}
	if len(s.Notes) != len(wantNotes) {
		t.Fatalf("Notes = %+v", s.Notes)
	}
	for i, w := range wantNotes {
		if s.Notes[i] != w {
			t.Errorf("Notes[%d] = %+v, want %+v", i, s.Notes[i], w)
		}
	}

	if s.DurationFormat != "H'h'm'm'" || s.DefaultOutput != "日報" {
		t.Errorf("DurationFormat = %q, DefaultOutput = %q", s.DurationFormat, s.DefaultOutput)
	}
}

func TestParseEmptyUsesDefaults(t *testing.T) {
	for _, input := range []string{"", "projects: []\n"} {
		s, err := Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse(%q): %v", input, err)
		}
		if s.DurationFormat != DefaultDurationFormat || len(s.Projects) != 0 || len(s.Notes) != 0 || s.DefaultOutput != "" {
			t.Errorf("Parse(%q) = %+v", input, s)
		}
	}
}

// 省略時の値と同じ項目は書かないので、何も設定していなければ空になる。
func TestMarshalDefaultIsEmpty(t *testing.T) {
	out, err := Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("Marshal(Default()) = %q", out)
	}
	s, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if s.DurationFormat != DefaultDurationFormat {
		t.Errorf("DurationFormat = %q", s.DurationFormat)
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"知らないキー":           "theme: dark\n",
		"プロジェクトの知らないキー":    "projects:\n  - code: a\n    label: b\n",
		"プロジェクトに code がない": "projects:\n  - name: a\n",
		"プロジェクトのコードが重なる":   "projects:\n  - code: a\n  - code: a\n    name: b\n",
		"期限が読めない":          "projects:\n  - code: a\n    until: 2026/12/31\n",
		"開始日が読めない":         "projects:\n  - code: a\n    start: 2026/04/01\n",
		"開始日が期限より後":        "projects:\n  - code: a\n    start: 2026-04-02\n    until: 2026-04-01\n",
		"メモ欄に name がない":    "notes:\n  - type: bool\n",
		"メモ欄の名前が重なる":       "notes:\n  - name: a\n  - name: a\n",
		"メモ欄の型が使えない":       "notes:\n  - name: a\n    type: choice\n",
		"長さの書式が使えない":       "duration_format: 'h:mm'\n",
		"YAML として壊れている":    "projects: [\n",
	}
	for name, input := range tests {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("%s: Parse succeeded, want error", name)
		}
	}
}

func TestInPeriod(t *testing.T) {
	d := func(m, day int) time.Time { return time.Date(2026, time.Month(m), day, 0, 0, 0, 0, time.UTC) }
	p := Project{Code: "a", Start: d(4, 1), Until: d(4, 30)}
	for date, want := range map[time.Time]bool{d(3, 31): false, d(4, 1): true, d(4, 30): true, d(5, 1): false} {
		if got := p.InPeriod(date); got != want {
			t.Errorf("InPeriod(%s) = %v", date.Format("2006-01-02"), got)
		}
	}
	// 開始日も期限もなければ、いつでも使える。
	if !(Project{Code: "b"}).InPeriod(d(1, 1)) {
		t.Error("期間のないプロジェクトが期間の外になった")
	}
}

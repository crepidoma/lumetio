package datefmt

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	// 月曜日。
	date := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		layout string
		want   string
	}{
		{"yyyyMMdd", "20260309"},
		{"yy/M/d", "26/3/9"},
		{"yyyy年MM月dd日(EEE)", "2026年03月09日(月)"},
		{"EEEE", "月曜日"},
		{"'report_'yyyyMMdd", "report_20260309"},
		// '' は ' を出す。
		{"yy''MM", "26'03"},
		{"'It''s 'd", "It's 9"},
		// 英字以外はそのまま出す。
		{"yyyy-MM-dd", "2026-03-09"},
		{"'固定'", "固定"},
		{"", ""},
	}
	for _, tt := range tests {
		f, err := Parse(tt.layout)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.layout, err)
			continue
		}
		if got := f.Format(date); got != tt.want {
			t.Errorf("Parse(%q).Format = %q, want %q", tt.layout, got, tt.want)
		}
	}
}

func TestFormatWeekdays(t *testing.T) {
	f, err := Parse("EEE")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"月", "火", "水", "木", "金", "土", "日"}
	for i, w := range want {
		date := time.Date(2026, 3, 9+i, 0, 0, 0, 0, time.UTC)
		if got := f.Format(date); got != w {
			t.Errorf("Format(%s) = %q, want %q", date.Format("2006-01-02"), got, w)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, layout := range []string{
		// 時刻は使えない。
		"yyyyMMddHHmm",
		"yyyyMMdd'_'HH",
		// 表にない並び。
		"yyy",
		"yyyyy",
		"MMM",
		"ddd",
		"E",
		"EEEEE",
		"D",
		// 英字をそのまま書きたいときは '…' で囲む。
		"report_yyyyMMdd",
		"'report",
	} {
		if _, err := Parse(layout); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", layout)
		}
	}
}

func TestParseDate(t *testing.T) {
	date := func(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }
	tests := []struct {
		layout, s string
		want      time.Time
	}{
		{"yyyy-MM-dd", "2026-03-09", date(2026, 3, 9)},
		{"yyyy/MM/dd", "2026/12/31", date(2026, 12, 31)},
		{"MM/dd/yyyy", "03/09/2026", date(2026, 3, 9)},
		{"dd/MM/yyyy", "09/03/2026", date(2026, 3, 9)},
		{"yyyy/M/d", "2026/3/9", date(2026, 3, 9)},
		{"yyyy/M/d", "2026/12/31", date(2026, 12, 31)},
		{"yyyyMMdd", "20260309", date(2026, 3, 9)},
		{"yy.MM.dd", "26.03.09", date(2026, 3, 9)},
		{"yyyy年M月d日", "2026年3月9日", date(2026, 3, 9)},
		{"'期限 'yyyy-MM-dd", "期限 2026-03-09", date(2026, 3, 9)},
		{"yyyy-MM-dd", "2028-02-29", date(2028, 2, 29)},
	}
	for _, tt := range tests {
		f, err := Parse(tt.layout)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.layout, err)
		}
		got, err := f.ParseDate(tt.s)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("%q で %q = %v, %v, want %v", tt.layout, tt.s, got, err, tt.want)
		}
	}
}

// 書式で書いたときと同じ形だけを読む。
func TestParseDateStrict(t *testing.T) {
	tests := []struct{ layout, s string }{
		{"yyyy-MM-dd", "2026-3-9"},
		{"yyyy-MM-dd", "2026/03/09"},
		{"yyyy-MM-dd", " 2026-03-09"},
		{"yyyy-MM-dd", "2026-03-09 "},
		{"yyyy-MM-dd", "2026-03-091"},
		{"yyyy/M/d", "2026/03/09"},
		{"yyyy/M/d", "2026/3/"},
		{"yyyy-MM-dd", "26-03-09"},
		{"yyyy-MM-dd", ""},
		// ない日。
		{"yyyy-MM-dd", "2026-02-29"},
		{"yyyy-MM-dd", "2026-13-01"},
		{"yyyy-MM-dd", "2026-00-10"},
		{"yyyy-MM-dd", "2026-04-31"},
	}
	for _, tt := range tests {
		f, err := Parse(tt.layout)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.layout, err)
		}
		if got, err := f.ParseDate(tt.s); err == nil {
			t.Errorf("%q で %q を %v と読んだ", tt.layout, tt.s, got)
		}
	}
}

func TestCheckReadable(t *testing.T) {
	for _, layout := range []string{"yyyy-MM-dd(EEE)", "yyyy-MM", "MM-dd", "yyyy-MM-dd-dd", "'固定'"} {
		f, err := Parse(layout)
		if err != nil {
			t.Fatalf("Parse(%q): %v", layout, err)
		}
		if f.CheckReadable() == nil {
			t.Errorf("%q を読める書式とした", layout)
		}
	}
}

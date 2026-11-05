package durfmt

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		layout  string
		minutes int
		want    string
	}{
		// 長さの書式の、代表的な例。
		{"H:mm", 90, "1:30"},
		{"H:mm", 75, "1:15"},
		{"H:mm", 50, "0:50"},
		{"m'm'", 90, "90m"},
		{"m'm'", 50, "50m"},
		{"H'h'm'm'", 90, "1h30m"},
		{"H'h'm'm'", 50, "0h50m"},
		{"H'h'", 90, "1.5h"},
		{"H'h'", 75, "1.25h"},
		{"H'h'", 50, "0.83h"},
		{"H'h'", 120, "2h"},

		{"HH:mm", 65, "01:05"},
		{"H:m", 65, "1:5"},
		{"HH'h'", 90, "01.5h"},
		{"H:mm", 0, "0:00"},
		{"H:mm", 1500, "25:00"},
		// 小数 2 桁で四捨五入する。10 分は 0.1666… 時間、20 分は 0.333… 時間。
		{"H", 10, "0.17"},
		{"H", 20, "0.33"},
		{"H", 1, "0.02"},
		// 英字以外はそのまま出す。
		{"H時間m分", 90, "1時間30分"},
		{"H.mm", 90, "1.30"},
		// '' は ' を出す。
		{"m''", 5, "5'"},
		{"'It''s 'm", 5, "It's 5"},
	}
	for _, tt := range tests {
		f, err := Parse(tt.layout)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.layout, err)
			continue
		}
		if got := f.Format(tt.minutes); got != tt.want {
			t.Errorf("Parse(%q).Format(%d) = %q, want %q", tt.layout, tt.minutes, got, tt.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, layout := range []string{
		"",
		":",
		"'H'",
		"HHH",
		"mmm",
		"H:mm:ss",
		"h:mm",
		"M'm'",
		"H'h",
	} {
		if _, err := Parse(layout); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", layout)
		}
	}
}

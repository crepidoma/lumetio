package timeexpr

import (
	"errors"
	"strings"
	"testing"
)

func at(h, m int) *int {
	v := h*60 + m
	return &v
}

func mustStored(t *testing.T, s string) Entry {
	t.Helper()
	e, err := ParseStored(s)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func joined(entries []Entry) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = e.String()
	}
	return strings.Join(parts, " ")
}

func TestParse(t *testing.T) {
	ctx := Context{
		Ranges: []Entry{mustStored(t, "09:30-11:30"), mustStored(t, "13:00-14:00")},
		Now:    at(17, 45),
	}
	tests := []struct {
		input string
		want  string
	}{
		{"17:00-18:00", "17:00-18:00"},
		{"9-10", "09:00-10:00"},
		{"930-1130", "09:30-11:30"},
		{"9:30-11", "09:30-11:00"},
		{"9 - 10", "09:00-10:00"},
		{"9- 10", "09:00-10:00"},
		{"13:00 -", "13:00-17:45"},
		{"30 -12:00", "30m 11:30-12:00"},
		{"17:00 ～ 18:00", "17:00-18:00"},
		{"17:00〜18:00", "17:00-18:00"},
		{"１７：００－１８：００", "17:00-18:00"},
		{"23-5", "23:00-05:00"},
		{"23:00-24:00", "23:00-00:00"},
		{"23:00-29:00", "23:00-05:00"},
		{"-12:00", "11:30-12:00"},
		{"-12", "11:30-12:00"},
		{"-15:00", "14:00-15:00"},
		{"13:00-", "13:00-17:45"},
		{"75", "75m"},
		{"1.5", "90m"},
		{".5", "30m"},
		{"0.33", "20m"},
		{"1:15", "75m"},
		{"30m", "30m"},
		{"1h", "60m"},
		{"1h15m", "75m"},
		{"1.5h", "90m"},
		{"1H15M", "75m"},
		{"24:00", "1440m"},
		{"30m 60m 17:00-18:00", "30m 60m 17:00-18:00"},
		{"30m,60m、17:00-18:00", "30m 60m 17:00-18:00"},
		{"30m　60m", "30m 60m"},
		{"16:00-16:30 -17:00", "16:00-16:30 16:30-17:00"},
		{"  ", ""},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input, ctx)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if s := joined(got); s != tt.want {
			t.Errorf("Parse(%q) = %q, want %q", tt.input, s, tt.want)
		}
	}
}

func TestParseMinutes(t *testing.T) {
	got, err := Parse("23-5 22:00-23:30", Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Minutes != 360 || got[1].Minutes != 90 {
		t.Errorf("minutes = %d, %d, want 360, 90", got[0].Minutes, got[1].Minutes)
	}
}

func TestParseErrors(t *testing.T) {
	ctx := Context{
		Ranges: []Entry{mustStored(t, "09:30-12:00")},
		Now:    at(17, 45),
	}
	inputs := []string{
		"10:00-10:00", // 開始と終了が同じ
		"0-24",        // 開始と終了が同じ (24:00 は 0:00)
		"-12:00",      // ちょうど 12:00 に終わる記録がある
		"-9:00",       // それより前に終わった記録がない
		"18:00-",      // まだその時刻になっていない
		"17:45-",      // 今と同じ
		"0",
		"0m",
		"0.001",
		"25:00",   // 長さとして 24 時間超え
		"1500",    // 分として 24 時間超え
		"48:00-1", // 時刻として読めない
		"9:60-10",
		"12345-1",
		"9-10-11",
		"-",
		"1:5",
		"1.5m",
		"abc",
		"1.",
	}
	for _, input := range inputs {
		_, err := Parse(input, ctx)
		var ie *InputError
		if !errors.As(err, &ie) {
			t.Errorf("Parse(%q) error = %v, want InputError", input, err)
		}
	}
}

func TestParseNowOnlyToday(t *testing.T) {
	_, err := Parse("13:00-", Context{})
	var ie *InputError
	if !errors.As(err, &ie) {
		t.Errorf("error = %v, want InputError", err)
	}
}

// どの日にも結び付かない入力では、その日の記録や今の時刻で決まる書き方を受け付けない。
func TestParseDetached(t *testing.T) {
	now := 15 * 60
	ctx := Context{Ranges: []Entry{newRange(9*60, 10*60)}, Now: &now, Detached: true}
	for _, input := range []string{"-12:00", "13:00-"} {
		_, err := Parse(input, ctx)
		var ie *InputError
		if !errors.As(err, &ie) {
			t.Errorf("Parse(%q) error = %v, want InputError", input, err)
		}
	}
	got, err := Parse("9-10 30m 1.5", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != newRange(9*60, 10*60) || got[1].Minutes != 30 || got[2].Minutes != 90 {
		t.Errorf("Parse = %+v", got)
	}
}

func TestParseStored(t *testing.T) {
	for _, s := range []string{"09:30-11:30", "23:00-05:00", "90m", "1440m"} {
		e, err := ParseStored(s)
		if err != nil {
			t.Errorf("ParseStored(%q) error: %v", s, err)
			continue
		}
		if e.String() != s {
			t.Errorf("ParseStored(%q).String() = %q", s, e.String())
		}
	}
	for _, s := range []string{"9-10", "24:00-01:00", "10:00-10:00", "90", "0m", "1441m", "1.5h"} {
		if _, err := ParseStored(s); err == nil {
			t.Errorf("ParseStored(%q) succeeded, want error", s)
		}
	}
}

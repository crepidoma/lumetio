package memo

import (
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/settings"
)

var defs = []settings.NoteDef{
	{Name: "在宅", Type: settings.NoteBool, CarryOver: true},
	{Name: "連絡", Type: settings.NoteText},
	{Name: "現場", Type: settings.NoteText, CarryOver: true},
}

func text(name, value string) dayfile.Note {
	return dayfile.Note{Name: name, Kind: dayfile.NoteText, Text: value}
}

func boolean(name string, value bool) dayfile.Note {
	return dayfile.Note{Name: name, Kind: dayfile.NoteBool, Bool: value}
}

func TestForNewDay(t *testing.T) {
	previous := &dayfile.Day{Notes: []dayfile.Note{
		boolean("在宅", true),
		text("連絡", "明日は休み"),
		text("現場", "本社"),
	}}
	got := ForNewDay(defs, previous)
	// 引き継ぐ欄だけ前回の値になり、引き継がない欄は空になる。
	want := []dayfile.Note{boolean("在宅", true), text("連絡", ""), text("現場", "本社")}
	if !slices.Equal(got, want) {
		t.Errorf("ForNewDay = %+v, want %+v", got, want)
	}
}

func TestForNewDayWithoutPrevious(t *testing.T) {
	got := ForNewDay(defs, nil)
	want := []dayfile.Note{boolean("在宅", false), text("連絡", ""), text("現場", "")}
	if !slices.Equal(got, want) {
		t.Errorf("ForNewDay = %+v, want %+v", got, want)
	}
}

// 前回の日にその欄がなければ空を引き継ぎ、型が違う値は持ち込まない。
func TestForNewDayMissingOrMismatched(t *testing.T) {
	previous := &dayfile.Day{Notes: []dayfile.Note{
		// 在宅を真偽値にする前に、文字列で書いていた。
		text("在宅", "はい"),
	}}
	got := ForNewDay(defs, previous)
	want := []dayfile.Note{boolean("在宅", false), text("連絡", ""), text("現場", "")}
	if !slices.Equal(got, want) {
		t.Errorf("ForNewDay = %+v, want %+v", got, want)
	}
}

func TestForDay(t *testing.T) {
	day := dayfile.Day{Notes: []dayfile.Note{
		text("昔の欄", "残っている値"),
		text("連絡", "なし"),
		// 定義は真偽値だが、ファイルにある値はそのまま出す。
		text("在宅", "はい"),
	}}
	got := ForDay(defs, day)
	want := []dayfile.Note{
		text("在宅", "はい"),
		text("連絡", "なし"),
		// 定義にあってファイルにない欄は空。前回の値は引き継がない。
		text("現場", ""),
		text("昔の欄", "残っている値"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("ForDay = %+v, want %+v", got, want)
	}
}

// 名前を変えた前の欄が空なら、定義のない欄として出さない。空の欄は書いていない欄と同じなので。
func TestForDayHidesEmptyUndefined(t *testing.T) {
	day := dayfile.Day{Notes: []dayfile.Note{text("ひとこと", ""), boolean("昔の真偽値", false), text("昔の欄", "残っている値")}}
	got := ForDay(defs, day)
	want := []dayfile.Note{boolean("在宅", false), text("連絡", ""), text("現場", ""), text("昔の欄", "残っている値")}
	if !slices.Equal(got, want) {
		t.Errorf("ForDay = %+v, want %+v", got, want)
	}
}

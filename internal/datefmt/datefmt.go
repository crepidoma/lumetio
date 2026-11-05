// Package datefmt は、日付を書式に従って文字列にし、その形の文字列を日付として読む。
// 記号は UTS#35 の日時の書式から、日付に使うものだけを選んだ。
package datefmt

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Format は解釈済みの書式。
type Format struct {
	parts []part
}

type part struct {
	// token が空なら literal をそのまま出す。
	token   string
	literal string
}

// tokens は使える記号の並び。
// 日次ファイルは時刻を持たないので、時刻の記号は入れない。
var tokens = []string{"yyyy", "yy", "MM", "M", "dd", "d", "EEEE", "EEE"}

var weekdays = [...]string{"日", "月", "火", "水", "木", "金", "土"}

// Parse は書式を解釈する。
func Parse(layout string) (Format, error) {
	var f Format
	for i := 0; i < len(layout); {
		c := layout[i]
		switch {
		case c == '\'':
			text, n, err := quoted(layout[i:])
			if err != nil {
				return Format{}, err
			}
			f.parts = append(f.parts, part{literal: text})
			i += n
		case isASCIILetter(c):
			n := 1
			for i+n < len(layout) && layout[i+n] == c {
				n++
			}
			run := layout[i : i+n]
			if !slices.Contains(tokens, run) {
				return Format{}, fmt.Errorf("書式の %q は使えません。英字は yyyy、yy、MM、M、dd、d、EEE、EEEE だけを使い、文字は '…' で囲みます", run)
			}
			f.parts = append(f.parts, part{token: run})
			i += n
		default:
			// 英字以外は記号ではないので、そのまま出す。
			j := i
			for j < len(layout) && layout[j] != '\'' && !isASCIILetter(layout[j]) {
				j++
			}
			f.parts = append(f.parts, part{literal: layout[i:j]})
			i = j
		}
	}
	return f, nil
}

// quoted は ' で始まる部分を読み、出す文字列と読んだバイト数を返す。
// ' を 2 つ続けると、' を 1 つ表す。
func quoted(s string) (string, int, error) {
	if strings.HasPrefix(s, "''") {
		return "'", 2, nil
	}
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		if s[i] != '\'' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '\'' {
			b.WriteByte('\'')
			i++
			continue
		}
		return b.String(), i + 1, nil
	}
	return "", 0, fmt.Errorf("書式の ' が閉じていません")
}

func isASCIILetter(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// Format は日付を書式に従って文字列にする。
func (f Format) Format(date time.Time) string {
	var b strings.Builder
	for _, p := range f.parts {
		switch p.token {
		case "":
			b.WriteString(p.literal)
		case "yyyy":
			b.WriteString(fmt.Sprintf("%04d", date.Year()))
		case "yy":
			b.WriteString(fmt.Sprintf("%02d", date.Year()%100))
		case "MM":
			b.WriteString(fmt.Sprintf("%02d", int(date.Month())))
		case "M":
			b.WriteString(strconv.Itoa(int(date.Month())))
		case "dd":
			b.WriteString(fmt.Sprintf("%02d", date.Day()))
		case "d":
			b.WriteString(strconv.Itoa(date.Day()))
		case "EEE":
			b.WriteString(weekdays[date.Weekday()])
		case "EEEE":
			b.WriteString(weekdays[date.Weekday()] + "曜日")
		}
	}
	return b.String()
}

// CheckReadable は、書式で日付を読めるかを確かめる。
// 年、月、日がそれぞれ 1 つずつ要る。曜日は、日付と食い違ったときにどちらを信じるか決められないので読まない。
func (f Format) CheckReadable() error {
	count := map[byte]int{}
	for _, p := range f.parts {
		switch p.token {
		case "":
		case "EEE", "EEEE":
			return fmt.Errorf("日付を読む書式に曜日 (%s) は使えません", p.token)
		default:
			count[p.token[0]]++
		}
	}
	if count['y'] != 1 || count['M'] != 1 || count['d'] != 1 {
		return fmt.Errorf("日付を読む書式には、年、月、日を 1 つずつ入れます")
	}
	return nil
}

// ParseDate は、書式で書いたときと同じ形の文字列だけを日付として読む。
// MM は必ず 2 桁、M は先頭に 0 を付けない 1〜2 桁。yy は 2000 年代とする。
func (f Format) ParseDate(s string) (time.Time, error) {
	if err := f.CheckReadable(); err != nil {
		return time.Time{}, err
	}
	var year, month, day int
	rest := s
	for _, p := range f.parts {
		if p.token == "" {
			if !strings.HasPrefix(rest, p.literal) {
				return time.Time{}, errNotDate
			}
			rest = rest[len(p.literal):]
			continue
		}
		var n int
		var ok bool
		switch p.token {
		case "yyyy":
			n, rest, ok = fixedDigits(rest, 4)
			year = n
		case "yy":
			n, rest, ok = fixedDigits(rest, 2)
			year = 2000 + n
		case "MM":
			n, rest, ok = fixedDigits(rest, 2)
			month = n
		case "M":
			n, rest, ok = shortDigits(rest)
			month = n
		case "dd":
			n, rest, ok = fixedDigits(rest, 2)
			day = n
		case "d":
			n, rest, ok = shortDigits(rest)
			day = n
		}
		if !ok {
			return time.Time{}, errNotDate
		}
	}
	if rest != "" {
		return time.Time{}, errNotDate
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// 2 月 30 日のようなない日は、time.Date が次の月へ繰り越すので、繰り越したかで見分ける。
	if month < 1 || month > 12 || date.Day() != day || int(date.Month()) != month {
		return time.Time{}, errNotDate
	}
	return date, nil
}

var errNotDate = errors.New("書式どおりの日付ではありません")

// fixedDigits は、s の先頭のちょうど n 桁の数字を読む。
func fixedDigits(s string, n int) (int, string, bool) {
	if len(s) < n {
		return 0, s, false
	}
	v, ok := number(s[:n])
	return v, s[n:], ok
}

// shortDigits は、s の先頭の、0 で始まらない 1〜2 桁の数字を読む。
func shortDigits(s string) (int, string, bool) {
	n := 0
	for n < len(s) && n < 2 && '0' <= s[n] && s[n] <= '9' {
		n++
	}
	if n == 0 || s[0] == '0' {
		return 0, s, false
	}
	v, _ := number(s[:n])
	return v, s[n:], true
}

func number(s string) (int, bool) {
	v := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		v = v*10 + int(s[i]-'0')
	}
	return v, true
}

// Package durfmt は、分で表した長さを書式に従って文字列にする。
// 記号は日付の書式 (yyyyMMdd HHmmss) にそろえる。
package durfmt

import (
	"fmt"
	"strconv"
	"strings"
)

// Format は解釈済みの書式。
type Format struct {
	parts []part
	hasH  bool
	hasM  bool
}

type part struct {
	// unit が 0 なら literal をそのまま出す。'H' か 'm' なら、その単位の値を width 桁にそろえて出す。
	unit    byte
	width   int
	literal string
}

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
			if c != 'H' && c != 'm' {
				return Format{}, fmt.Errorf("書式の %q は使えません。英字は H と m だけを使い、文字は '…' で囲みます", layout[i:i+n])
			}
			if n > 2 {
				return Format{}, fmt.Errorf("書式の %q は使えません。%c か %c%c で書きます", layout[i:i+n], c, c, c)
			}
			f.parts = append(f.parts, part{unit: c, width: n})
			if c == 'H' {
				f.hasH = true
			} else {
				f.hasM = true
			}
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
	if !f.hasH && !f.hasM {
		return Format{}, fmt.Errorf("書式に H も m もありません")
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

// Format は分を書式に従って文字列にする。minutes は 0 以上とする。
func (f Format) Format(minutes int) string {
	var hours, mins string
	switch {
	case f.hasH && f.hasM:
		hours = strconv.Itoa(minutes / 60)
		mins = strconv.Itoa(minutes % 60)
	case f.hasH:
		// 時間がいちばん小さい単位なので、余りの分を小数で出す。
		hours = decimalHours(minutes)
	default:
		// H がないので、分に時間の分も入る。
		mins = strconv.Itoa(minutes)
	}

	var b strings.Builder
	for _, p := range f.parts {
		switch p.unit {
		case 'H':
			b.WriteString(pad(hours, p.width))
		case 'm':
			b.WriteString(pad(mins, p.width))
		default:
			b.WriteString(p.literal)
		}
	}
	return b.String()
}

// decimalHours は分を時間にし、小数 2 桁で四捨五入して、末尾の 0 を落とす。
func decimalHours(minutes int) string {
	hundredths := minutes * 100 / 60
	if minutes*100%60*2 >= 60 {
		hundredths++
	}
	s := strconv.Itoa(hundredths / 100)
	frac := fmt.Sprintf("%02d", hundredths%100)
	frac = strings.TrimRight(frac, "0")
	if frac != "" {
		s += "." + frac
	}
	return s
}

// pad は整数部を width 桁にそろえる。小数部はそのまま残す。
func pad(s string, width int) string {
	intPart, _, _ := strings.Cut(s, ".")
	if n := width - len(intPart); n > 0 {
		return strings.Repeat("0", n) + s
	}
	return s
}

// Package memo は、その日のメモ欄に出す値を決める。
package memo

import (
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/settings"
)

// ForNewDay は、まだ日次ファイルのない日のメモ欄の値を、定義の順に作る。
// 日次ファイルを作るときは、これを書く (空の欄は書かれない)。
// メモ欄に触らなかった日も、引き継いだ値が途切れないようにするため。
// previous は、その日より前で日次ファイルがある一番近い日。なければ nil。
func ForNewDay(defs []settings.NoteDef, previous *dayfile.Day) []dayfile.Note {
	notes := make([]dayfile.Note, 0, len(defs))
	for _, def := range defs {
		n := empty(def)
		if def.CarryOver && previous != nil {
			// 前回の日にその欄がなければ、空を引き継ぐ。
			// 型が今の定義と違う値は、新しい日に持ち込まない。
			if p, ok := find(previous.Notes, def.Name); ok && p.Kind == n.Kind {
				n = p
			}
		}
		notes = append(notes, n)
	}
	return notes
}

// ForDay は、日次ファイルのある日のメモ欄の値を返す。
// ファイルの値をそのまま使い、前回の値は引き継がない。その日のファイルが正なので。
// 定義にある欄を定義の順に並べ、定義にあってファイルにない欄は空にする。
// 定義にない欄 (後で消した欄など) は、その後ろにファイルに書かれた順で並べる。
// 定義にない空の欄は、書いていない欄と同じなので出さない。
func ForDay(defs []settings.NoteDef, day dayfile.Day) []dayfile.Note {
	notes := make([]dayfile.Note, 0, len(defs)+len(day.Notes))
	defined := map[string]bool{}
	for _, def := range defs {
		defined[def.Name] = true
		if n, ok := find(day.Notes, def.Name); ok {
			notes = append(notes, n)
			continue
		}
		notes = append(notes, empty(def))
	}
	for _, n := range day.Notes {
		if !defined[n.Name] && !n.Empty() {
			notes = append(notes, n)
		}
	}
	return notes
}

func empty(def settings.NoteDef) dayfile.Note {
	if def.Type == settings.NoteBool {
		return dayfile.Note{Name: def.Name, Kind: dayfile.NoteBool}
	}
	return dayfile.Note{Name: def.Name, Kind: dayfile.NoteText}
}

func find(notes []dayfile.Note, name string) (dayfile.Note, bool) {
	for _, n := range notes {
		if n.Name == name {
			return n, true
		}
	}
	return dayfile.Note{}, false
}

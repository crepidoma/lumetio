package app

import (
	"slices"

	"github.com/crepidoma/lumetio/internal/settings"
)

// NoteDefs は、メモ欄の定義を settings.yaml の並び (画面に出す順) で返す。
func (a *App) NoteDefs() ([]settings.NoteDef, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	return st.Notes, nil
}

// SaveNoteDefs はメモ欄の定義を書く。名前は前後の空白を除き、NFC にそろえる。
// 日次ファイルはどの日も書き換えない。過去の日を書いたときのまま開けるようにするため。
// base は設定画面で読んだときの定義。他で定義が変わっていたら書かずに ErrConflict を返す。
// 比べるのは定義だけにし、settings.yaml のほかの項目は、その時点の中身のまま残す。
func (a *App) SaveNoteDefs(base, defs []settings.NoteDef) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]settings.NoteDef, 0, len(defs))
	for _, d := range defs {
		d.Name = cleanName(d.Name)
		out = append(out, d)
	}
	if err := checkNoteDefs(out); err != nil {
		return err
	}
	return a.updateSettings(
		func(st settings.Settings) bool { return slices.Equal(st.Notes, base) },
		func(st *settings.Settings) { st.Notes = out },
	)
}

// checkNoteDefs は、settings.yaml として読めない定義 (名前が空、同じ名前、知らない型) を入力の誤りにする。
func checkNoteDefs(defs []settings.NoteDef) error {
	names := map[string]bool{}
	for i, d := range defs {
		if d.Name == "" {
			return inputErrorf("%d 行目の名前が空です", i+1)
		}
		if names[d.Name] {
			return inputErrorf("名前 %q が重なっています", d.Name)
		}
		names[d.Name] = true
		if d.Type != settings.NoteText && d.Type != settings.NoteBool {
			return inputErrorf("%q の型 %q は使えません", d.Name, d.Type)
		}
	}
	return nil
}

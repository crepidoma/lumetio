package app

import (
	"github.com/crepidoma/lumetio/internal/settings"
)

// SaveDurationFormat は長さの表記を書く。空なら省略時の表記にする。
// base は設定画面を開いたときの表記。他で変わっていたら書かずに ErrConflict を返す。
// 比べるのは長さの表記だけにし、settings.yaml のほかの項目は、その時点の中身のまま残す。
func (a *App) SaveDurationFormat(base, layout string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	layout, _, err := settings.ParseDurationFormat(layout)
	if err != nil {
		return &InputError{Message: err.Error()}
	}
	return a.updateSettings(
		func(st settings.Settings) bool { return st.DurationFormat == base },
		func(st *settings.Settings) { st.DurationFormat = layout },
	)
}

// updateSettings は settings.yaml の 1 つの項目を書く。a.mu を持ったまま呼ぶ。
// same は、今のファイルのその項目が、画面で読んだときと同じか。違えば書かずに ErrConflict を返す。
// 比べるのはその項目だけにし、settings.yaml のほかの項目は、その時点の中身のまま残す。
// 項目ごとに保存する設定画面で、ほかの項目の変更を上書きしないため。
func (a *App) updateSettings(same func(settings.Settings) bool, apply func(*settings.Settings)) error {
	st, ver, err := a.store.LoadSettings()
	if err != nil {
		return err
	}
	if !same(st) {
		return ErrConflict
	}
	apply(&st)
	_, err = a.store.SaveSettings(st, ver)
	return err
}

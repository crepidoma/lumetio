package app

import (
	"slices"

	"github.com/crepidoma/lumetio/internal/settings"
)

// Tags は、タグの一覧を settings.yaml の並び (候補の順) で返す。
func (a *App) Tags() ([]string, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	return st.Tags, nil
}

// SaveTags はタグの一覧を書く。名前は前後の空白を除き、NFC にそろえる。
// 日次ファイルと items.yaml のタグは書き換えない。過去の日を書いたときのまま開けるようにするため。
// base は設定画面で読んだときの一覧。他で一覧が変わっていたら書かずに ErrConflict を返す。
// 比べるのは一覧だけにし、settings.yaml のほかの項目は、その時点の中身のまま残す。
func (a *App) SaveTags(base, tags []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for i, t := range tags {
		t = cleanName(t)
		if t == "" {
			return inputErrorf("%d 行目の名前が空です", i+1)
		}
		if seen[t] {
			return inputErrorf("名前 %q が重なっています", t)
		}
		seen[t] = true
		out = append(out, t)
	}
	return a.updateSettings(
		func(st settings.Settings) bool { return slices.Equal(st.Tags, base) },
		func(st *settings.Settings) { st.Tags = out },
	)
}

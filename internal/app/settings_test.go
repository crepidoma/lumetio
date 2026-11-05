package app

import (
	"errors"
	"os"
	"testing"
)

func (f *fixture) settingsFile() string {
	f.t.Helper()
	data, err := os.ReadFile(f.store.SettingsPath())
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

func TestSaveDurationFormat(t *testing.T) {
	f := newFixture(t)

	// 長さの表記だけを変え、ほかの項目はそのまま残す。
	if err := f.app.SaveDurationFormat("H:mm", "H'h'"); err != nil {
		t.Fatal(err)
	}
	st, _, err := f.store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if st.DurationFormat != "H'h'" || len(st.Projects) != 2 || len(st.Notes) != 2 || st.DefaultOutput != "日報" {
		t.Errorf("settings = %+v", st)
	}

	// 空にすると省略時の表記に戻り、settings.yaml からは消える。
	if err := f.app.SaveDurationFormat("H'h'", " "); err != nil {
		t.Fatal(err)
	}
	if got := f.settingsFile(); got != settingsYAML {
		t.Errorf("settings.yaml:\n%s\nwant:\n%s", got, settingsYAML)
	}

	if err := f.app.SaveDurationFormat("H:mm", "d"); !IsInputError(err) {
		t.Errorf("読めない書式: err = %v, want input error", err)
	}

	// 開いた後に他で長さの表記が変わっていたら書かない。
	if err := f.app.SaveDurationFormat("m'm'", "H'h'"); !errors.Is(err, ErrConflict) {
		t.Errorf("他で変わった: err = %v, want ErrConflict", err)
	}
	if got := f.settingsFile(); got != settingsYAML {
		t.Errorf("衝突したのに書いた:\n%s", got)
	}
}

package gui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/timeexpr"
)

// 種類で、画面が入力の誤りと衝突と内部の失敗を見分けられる。
func TestMarshalError(t *testing.T) {
	_, inputErr := timeexpr.Parse("あとで", timeexpr.Context{})
	tests := []struct {
		err  error
		want string
	}{
		{inputErr, `{"kind":"input"}`},
		{&app.InputError{Message: "x"}, `{"kind":"input"}`},
		{fmt.Errorf("保存: %w", app.ErrConflict), `{"kind":"conflict"}`},
		{errors.New("読めない"), `{"kind":"internal"}`},
	}
	for _, tt := range tests {
		if got := string(MarshalError(tt.err)); got != tt.want {
			t.Errorf("%v: %s, want %s", tt.err, got, tt.want)
		}
	}
}

package gui

import (
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/tasklist"
)

func TestSearch(t *testing.T) {
	var candidates []app.Candidate
	for _, title := range []string{"ＡＢＣの調査", "abc 資料", "カタカナ", "ひらがな", "DEV-１２３ 対応"} {
		candidates = append(candidates, app.Candidate{Title: title})
	}
	tests := []struct {
		query string
		want  []string
	}{
		// 全角と半角、大文字と小文字を区別しない。
		{"abc", []string{"ＡＢＣの調査", "abc 資料"}},
		{"ＡＢＣ", []string{"ＡＢＣの調査", "abc 資料"}},
		{"dev-123", []string{"DEV-１２３ 対応"}},
		// ひらがなとカタカナは区別する。
		{"かたかな", nil},
		{" 資料 ", []string{"abc 資料"}},
	}
	for _, tt := range tests {
		var got []string
		for _, c := range search(candidates, tt.query) {
			got = append(got, c.Title)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%q: %v, want %v", tt.query, got, tt.want)
		}
	}
}

// 直近のタスクのプロジェクトは、＋で記録するときの値 (Base) を出す。左ペインで見る値と、記録される値をそろえるため。
func TestNewTaskListsRecentProject(t *testing.T) {
	l := app.TaskLists{
		Recent: []tasklist.Recent{{
			Last: dayfile.Task{Title: "調査", Project: dayfile.Project{Code: "OLD-1"}},
			Base: dayfile.Task{Title: "調査", Project: dayfile.Project{Code: "DEV-1"}},
		}},
	}
	got := newTaskLists(l)
	if len(got.Recent) != 1 || got.Recent[0].Project == nil || got.Recent[0].Project.Code != "DEV-1" {
		t.Errorf("Recent = %+v", got.Recent)
	}
}

// 直近のタスクからは、よく使うタスクとその日に予定の出るタスクを除く。
func TestNewTaskLists(t *testing.T) {
	l := app.TaskLists{
		Recent: []tasklist.Recent{
			{Last: dayfile.Task{Title: "調査"}},
			{Last: dayfile.Task{Title: "問い合わせ対応"}},
			{Last: dayfile.Task{Title: "朝会"}},
			{Last: dayfile.Task{Title: "月例報告"}},
		},
		Favorites:    []items.Item{{Title: "問い合わせ対応", Favorite: true}},
		RepeatingOn:  []items.Item{{Title: "朝会"}},
		RepeatingOff: []items.Item{{Title: "月例報告"}},
		Recorded:     []string{"朝会"},
	}
	got := newTaskLists(l)
	var recent []string
	for _, r := range got.Recent {
		recent = append(recent, r.Title)
	}
	// 今日に予定の出ない定期的なタスクは、折りたたまれるので直近のタスクに残す。
	if !slices.Equal(recent, []string{"調査", "月例報告"}) {
		t.Errorf("Recent = %v", recent)
	}
	if !got.Favorites[0].Favorite || !got.RepeatingOn[0].Recorded || got.Favorites[0].Recorded {
		t.Errorf("lists = %+v", got)
	}
}

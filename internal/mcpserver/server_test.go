package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 2026-11-05 は木曜日。
var now = time.Date(2026, 11, 5, 15, 20, 0, 0, time.Local)

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const itemsYAML = "- title: 朝会\n  time: ['09:00-09:15']\n  repeat:\n    freq: daily\n"

// newRoot は、テスト用のデータフォルダを作る。
func newRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "settings.yaml", "projects:\n  - code: DEV-001\n    name: 新機能\nnotes:\n  - name: 在宅\n    type: bool\n  - name: 連絡\ndefault_output: 日報\n")
	write(t, root, "items.yaml", itemsYAML)
	write(t, root, "templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}} {{.time}}\n{{end}}")
	write(t, root, "templates/日報.set.yaml", "kind: set\noutputs: [日報テキスト]\n")
	return root
}

// connect は、テスト用のデータフォルダで動くサーバにつないだクライアントを返す。
func connect(t *testing.T) (*mcp.ClientSession, string) {
	t.Helper()
	return connectTo(t, newRoot(t))
}

// connectTo は、root をデータフォルダにして動くサーバにつないだクライアントと、出力先を返す。
func connectTo(t *testing.T, root string) (*mcp.ClientSession, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out")

	a := app.New(store.New(root), out, func() time.Time { return now })
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := New(a, "test").Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, out
}

// call はツールを呼び、結果の構造化された中身を v に読む。
// ツールのエラーなら、その文を返す。
func call(t *testing.T, cs *mcp.ClientSession, name string, args any, v any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		return res.Content[0].(*mcp.TextContent).Text
	}
	if v != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, v); err != nil {
			t.Fatal(err)
		}
	}
	return ""
}

func TestListTools(t *testing.T) {
	cs, _ := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	want := []string{"add_records", "export_output_set", "get_days", "list_projects", "list_tags", "list_tasks", "remove_record", "render_output", "set_notes", "skip_plans", "update_record"}
	if !slices.Equal(names, want) {
		t.Errorf("tools = %v", names)
	}
}

// 記録して、読んで、出力するまでの一連の流れ。
// 直近のタスクが items.yaml にあれば、add_records で記録されるのと同じ items.yaml の値と、いつもの時間を返す。
// 最後の記録の値を返すと、AI が見る値と実際に記録される値がずれるため。
func TestListTasksRecentUsesItems(t *testing.T) {
	root := newRoot(t)
	write(t, root, "items.yaml", itemsYAML+"- title: 調査\n  project:\n    code: DEV-001\n    name: 新機能\n  tags: [設計]\n  time: [60m]\n")
	write(t, root, "days/2026/2026-11-04.yaml", "date: 2026-11-04\ntasks:\n  - title: 調査\n    project:\n      code: OLD-01\n  - title: 雑談\n    project:\n      code: ETC-01\n    exclude: true\n")
	cs, _ := connectTo(t, root)

	var lists taskListsOutput
	if msg := call(t, cs, "list_tasks", map[string]any{}, &lists); msg != "" {
		t.Fatal(msg)
	}
	got := map[string]taskOutput{}
	for _, r := range lists.Recent {
		got[r.Title] = r
	}
	if r := got["調査"]; r.Project == nil || r.Project.Code != "DEV-001" || !slices.Equal(r.Tags, []string{"設計"}) || len(r.UsualTime) != 1 {
		t.Errorf("調査 = %+v", r)
	}
	// items.yaml にないタスクは、最後の記録から写す。
	if r := got["雑談"]; r.Project == nil || r.Project.Code != "ETC-01" || !r.Exclude || len(r.UsualTime) != 0 {
		t.Errorf("雑談 = %+v", r)
	}
}

func TestRecordAndOutput(t *testing.T) {
	cs, out := connect(t)

	var days daysOutput
	if msg := call(t, cs, "get_days", map[string]any{}, &days); msg != "" {
		t.Fatal(msg)
	}
	if len(days.Days) != 1 || days.Days[0].Date != "2026-11-05" || len(days.Days[0].Plans) != 1 || days.Days[0].Exists {
		t.Fatalf("get_days = %+v", days)
	}

	if msg := call(t, cs, "add_records", map[string]any{
		"records": []any{
			map[string]any{"title": "朝会"},
			map[string]any{"title": "調査", "time": "-12:00", "project": map[string]any{"code": "DEV-001"}},
		},
	}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "set_notes", map[string]any{"notes": map[string]any{"在宅": true, "連絡": "なし"}}, nil); msg != "" {
		t.Fatal(msg)
	}
	if msg := call(t, cs, "update_record", map[string]any{"title": "調査", "note": "続きは明日"}, nil); msg != "" {
		t.Fatal(msg)
	}

	if msg := call(t, cs, "get_days", map[string]any{"from": "2026-11-05"}, &days); msg != "" {
		t.Fatal(msg)
	}
	d := days.Days[0]
	if len(d.Plans) != 0 || len(d.Records) != 2 || d.TotalMinutes != 180 {
		t.Errorf("day = %+v", d)
	}
	if r := d.Records[1]; r.Title != "調査" || r.Project == nil || r.Project.Name != "新機能" || r.Note != "続きは明日" || !slices.Equal(r.Time, []string{"09:15-12:00"}) {
		t.Errorf("調査 = %+v", r)
	}
	if len(d.Notes) != 2 || d.Notes[0].Value != true || d.Notes[0].Type != "bool" || d.Notes[1].Value != "なし" {
		t.Errorf("notes = %+v", d.Notes)
	}

	var rendered renderOutputResult
	if msg := call(t, cs, "render_output", map[string]any{}, &rendered); msg != "" {
		t.Fatal(msg)
	}
	if len(rendered.Outputs) != 1 || rendered.Outputs[0].Text != "朝会 0:15\n調査 2:45\n" {
		t.Errorf("render_output = %+v", rendered)
	}

	var exported exportResult
	if msg := call(t, cs, "export_output_set", map[string]any{}, &exported); msg != "" {
		t.Fatal(msg)
	}
	if _, err := os.Stat(filepath.Join(out, "report_20261105.txt")); err != nil {
		t.Error(err)
	}
	// 2 回目は、扱いを指定しないとエラーになり、あるファイルを伝える。
	if msg := call(t, cs, "export_output_set", map[string]any{}, nil); !strings.Contains(msg, "report_20261105.txt") {
		t.Errorf("2 回目: %q", msg)
	}
	if msg := call(t, cs, "export_output_set", map[string]any{"on_conflict": "overwrite"}, &exported); msg != "" || !slices.Equal(exported.Written, []string{"report_20261105.txt"}) {
		t.Errorf("上書き: %q, %+v", msg, exported)
	}
}

func TestToolErrors(t *testing.T) {
	cs, _ := connect(t)
	tests := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"get_days", map[string]any{"from": "2026/11/05"}, "YYYY-MM-DD"},
		{"get_days", map[string]any{"from": "2026-10-01", "to": "2026-11-05"}, "31 日"},
		{"get_days", map[string]any{"from": "2026-11-05", "to": "2026-11-04"}, "from 以降"},
		{"add_records", map[string]any{"records": []any{map[string]any{"title": "a", "time": "あとで"}}}, "あとで"},
		{"set_notes", map[string]any{"notes": map[string]any{"在宅勤務": true}}, "在宅 (真偽値)"},
		{"skip_plans", map[string]any{"titles": []any{"ない予定"}}, "予定にありません"},
		{"remove_record", map[string]any{"title": "ない"}, "記録はありません"},
		{"render_output", map[string]any{"name": "ない"}, "日報テキスト"},
		{"export_output_set", map[string]any{"on_conflict": "skip"}, "overwrite、rename、ask"},
	}
	for _, tt := range tests {
		if msg := call(t, cs, tt.tool, tt.args, nil); !strings.Contains(msg, tt.want) {
			t.Errorf("%s %v: %q, want containing %q", tt.tool, tt.args, msg, tt.want)
		}
	}
}

// タグの一覧を、settings.yaml の並びで返す。一覧がなければ空の一覧を返す。
func TestListTags(t *testing.T) {
	root := newRoot(t)
	cs, _ := connectTo(t, root)
	var out tagsOutput
	if msg := call(t, cs, "list_tags", map[string]any{}, &out); msg != "" {
		t.Fatal(msg)
	}
	if out.Tags == nil || len(out.Tags) != 0 {
		t.Errorf("tags = %#v", out.Tags)
	}

	write(t, root, "settings.yaml", "tags: [ふりかえり, 調査]\n")
	if msg := call(t, cs, "list_tags", map[string]any{}, &out); msg != "" {
		t.Fatal(msg)
	}
	if !slices.Equal(out.Tags, []string{"ふりかえり", "調査"}) {
		t.Errorf("tags = %q", out.Tags)
	}
}

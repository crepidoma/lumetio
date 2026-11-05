// Package mcpserver は、lumetio の操作を MCP のツールとして公開する。
// 決まりはすべて app にあり、ここは引数と結果の形を変えるだけにする。
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/output"
	"github.com/crepidoma/lumetio/internal/yamlfile"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxDays は get_days で一度に読める日数。
// 長い期間を読むと、AI に渡す結果が大きくなりすぎるため。
const maxDays = 31

// New は、a を操作する MCP サーバを作る。
func New(a *app.App, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "lumetio", Version: version}, &mcp.ServerOptions{
		Instructions: "日々のタスクの記録 (日報と工数) を読み書きする。" +
			"日付は YYYY-MM-DD で、省略すると今日。" +
			"記録は、その日のタスク名で指す。" +
			"記録を足す前に list_tasks で既存のタスク名を確かめ、同じタスクには同じ名前を使う。",
	})
	h := handlers{app: a}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Description: "その日を起点に、直近のタスク (14 日以内に記録したもの)、よく使うタスク、定期的なタスクを返す。各タスクのプロジェクト、タグ、計上しない、いつもの時間を含む。",
	}, h.listTasks)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_projects",
		Description: "その日に使えるプロジェクト (コードと名前) を返す。期間の外 (開始前か期限切れ) のものは返さない。",
	}, h.listProjects)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_days",
		Description: fmt.Sprintf("期間の各日の記録、予定 (まだ記録にしていない定期的なタスク)、スキップした予定、メモ欄、プロジェクトごとの合計を返す。期間は最大 %d 日。省略すると今日 1 日。", maxDays),
	}, h.getDays)
	mcp.AddTool(s, &mcp.Tool{
		Name: "add_records",
		Description: "その日の記録にタスクをまとめて足す。1 つでも誤りがあれば何も書かない。" +
			"予定と同じ名前のタスクを足すと、予定は記録になる (実施済)。" +
			"時間を省略すると、いつもの時間で入る。" +
			"プロジェクトなどを省略すると、タスクの設定か、そのタスクの一番新しい記録から写す。" +
			"その日にもう同じタスクの記録があれば、時間だけを足す。",
	}, h.addRecords)
	mcp.AddTool(s, &mcp.Tool{
		Name: "update_record",
		Description: "その日の記録を直す。指定した項目だけを変える。" +
			"完了 (done) は、そのタスクを最後に記録した日の記録に付ける。完了にしたタスクは、直近のタスクに出なくなる。",
	}, h.updateRecord)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_record",
		Description: "その日の記録を消す。取り消しはできない。",
	}, h.removeRecord)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "skip_plans",
		Description: "その日の予定をまとめてスキップする (その日だけ消す)。休みの日や中止になった会議に使う。",
	}, h.skipPlans)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_notes",
		Description: "その日のメモ欄をまとめて書く。書けるのは、get_days の notes に出る、定義のある欄だけ。値は欄の型 (text は文字列、bool は true か false) に合わせる。",
	}, h.setNotes)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "render_output",
		Description: "その日の日報テキストや工数 CSV などの出力を作って、中身を返す。ファイルには書かない。名前は出力か出力セットの名前で、省略すると既定の出力。",
	}, h.renderOutput)
	mcp.AddTool(s, &mcp.Tool{
		Name: "export_output_set",
		Description: "その日の出力セットを作って、出力先のフォルダに書く。名前を省略すると既定の出力。" +
			"出力先に同じ名前のファイルがあれば、on_conflict を省略したときは何も書かずにエラーを返す。ユーザーに上書きか別名かを尋ねてから、on_conflict を指定して呼び直す。",
	}, h.exportOutputSet)
	return s
}

type handlers struct {
	app *app.App
}

func (h handlers) date(s string) (time.Time, error) {
	if s == "" {
		return h.app.Today(), nil
	}
	d, err := time.Parse(yamlfile.DateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("日付 %q は YYYY-MM-DD で書きます", s)
	}
	return d, nil
}

// toolError は、AI が次に何をすればよいか分かるように、誤りの文を整える。
func toolError(err error) error {
	if errors.Is(err, app.ErrConflict) {
		return errors.New("読んだ後にファイルが他 (アプリの画面や別の PC) で変更されたため、書きませんでした。読み直してから、もう一度呼んでください")
	}
	return err
}

func (h handlers) listTasks(_ context.Context, _ *mcp.CallToolRequest, in dateInput) (*mcp.CallToolResult, taskListsOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, taskListsOutput{}, err
	}
	lists, err := h.app.Tasks(date)
	if err != nil {
		return nil, taskListsOutput{}, toolError(err)
	}
	return nil, newTaskListsOutput(lists), nil
}

func (h handlers) listProjects(_ context.Context, _ *mcp.CallToolRequest, in dateInput) (*mcp.CallToolResult, projectsOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, projectsOutput{}, err
	}
	ps, err := h.app.Projects(date)
	if err != nil {
		return nil, projectsOutput{}, toolError(err)
	}
	out := projectsOutput{Projects: []projectOutput{}}
	for _, p := range ps {
		po := projectOutput{Code: p.Code, Name: p.Name}
		if !p.Until.IsZero() {
			po.Until = p.Until.Format(yamlfile.DateLayout)
		}
		out.Projects = append(out.Projects, po)
	}
	return nil, out, nil
}

func (h handlers) getDays(_ context.Context, _ *mcp.CallToolRequest, in getDaysInput) (*mcp.CallToolResult, daysOutput, error) {
	from, err := h.date(in.From)
	if err != nil {
		return nil, daysOutput{}, err
	}
	to := from
	if in.To != "" {
		if to, err = h.date(in.To); err != nil {
			return nil, daysOutput{}, err
		}
	}
	if to.Before(from) {
		return nil, daysOutput{}, errors.New("to は from 以降の日付にします")
	}
	if n := int(to.Sub(from).Hours()/24) + 1; n > maxDays {
		return nil, daysOutput{}, fmt.Errorf("期間は %d 日までです (%d 日を指定しました)", maxDays, n)
	}
	out := daysOutput{Days: []dayOutput{}}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		view, err := h.app.Day(d)
		if err != nil {
			return nil, daysOutput{}, toolError(err)
		}
		out.Days = append(out.Days, newDayOutput(view))
	}
	return nil, out, nil
}

func (h handlers) addRecords(_ context.Context, _ *mcp.CallToolRequest, in addRecordsInput) (*mcp.CallToolResult, writeOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, writeOutput{}, err
	}
	var recs []app.NewRecord
	for _, r := range in.Records {
		recs = append(recs, app.NewRecord{
			Title:   r.Title,
			Time:    r.Time,
			Project: r.Project.toApp(),
			Tags:    r.Tags,
			Note:    r.Note,
			Exclude: r.Exclude,
		})
	}
	warnings, err := h.app.AddRecords(date, recs)
	if err != nil {
		return nil, writeOutput{}, toolError(err)
	}
	return nil, newWriteOutput(warnings), nil
}

func (h handlers) updateRecord(_ context.Context, _ *mcp.CallToolRequest, in updateRecordInput) (*mcp.CallToolResult, writeOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, writeOutput{}, err
	}
	warnings, err := h.app.UpdateRecord(date, in.Title, app.RecordPatch{
		Time:    in.Time,
		Project: in.Project.toApp(),
		Tags:    in.Tags,
		Note:    in.Note,
		Exclude: in.Exclude,
		Done:    in.Done,
	})
	if err != nil {
		return nil, writeOutput{}, toolError(err)
	}
	return nil, newWriteOutput(warnings), nil
}

func (h handlers) removeRecord(_ context.Context, _ *mcp.CallToolRequest, in titleInput) (*mcp.CallToolResult, writeOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, writeOutput{}, err
	}
	if err := h.app.RemoveRecord(date, in.Title); err != nil {
		return nil, writeOutput{}, toolError(err)
	}
	return nil, newWriteOutput(nil), nil
}

func (h handlers) skipPlans(_ context.Context, _ *mcp.CallToolRequest, in skipPlansInput) (*mcp.CallToolResult, writeOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, writeOutput{}, err
	}
	if err := h.app.SkipPlans(date, in.Titles); err != nil {
		return nil, writeOutput{}, toolError(err)
	}
	return nil, newWriteOutput(nil), nil
}

func (h handlers) setNotes(_ context.Context, _ *mcp.CallToolRequest, in setNotesInput) (*mcp.CallToolResult, writeOutput, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, writeOutput{}, err
	}
	// 誤りの知らせが呼ぶたびに変わらないよう、名前の順に書く。
	names := make([]string, 0, len(in.Notes))
	for name := range in.Notes {
		names = append(names, name)
	}
	slices.Sort(names)
	values := make([]app.NoteValue, 0, len(names))
	for _, name := range names {
		values = append(values, app.NoteValue{Name: name, Value: in.Notes[name]})
	}
	if err := h.app.SetNotes(date, values); err != nil {
		return nil, writeOutput{}, toolError(err)
	}
	return nil, newWriteOutput(nil), nil
}

func (h handlers) renderOutput(_ context.Context, _ *mcp.CallToolRequest, in outputInput) (*mcp.CallToolResult, renderOutputResult, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, renderOutputResult{}, err
	}
	r, err := h.app.Render(date, in.Name)
	if err != nil {
		return nil, renderOutputResult{}, toolError(err)
	}
	return nil, newRenderOutputResult(r), nil
}

func (h handlers) exportOutputSet(_ context.Context, _ *mcp.CallToolRequest, in exportInput) (*mcp.CallToolResult, exportResult, error) {
	date, err := h.date(in.Date)
	if err != nil {
		return nil, exportResult{}, err
	}
	conflict, ok := output.ParseConflict(in.OnConflict)
	if !ok {
		return nil, exportResult{}, fmt.Errorf("on_conflict の %q は使えません。overwrite、rename、ask のどれかを書きます", in.OnConflict)
	}
	written, r, err := h.app.Export(date, in.Name, conflict)
	if err != nil {
		return nil, exportResult{}, toolError(err)
	}
	return nil, exportResult{Written: written, Warnings: nonNil(r.Warnings), Notices: nonNil(r.Skipped)}, nil
}

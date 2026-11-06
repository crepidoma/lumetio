package mcpserver

import (
	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/effort"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

type dateInput struct {
	Date string `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
}

// noInput は、引数のないツールの入力。
type noInput struct{}

type getDaysInput struct {
	From string `json:"from,omitempty" jsonschema:"期間の始まり (YYYY-MM-DD)。省略すると今日"`
	To   string `json:"to,omitempty" jsonschema:"期間の終わり (YYYY-MM-DD、この日を含む)。省略すると from と同じ日"`
}

type projectInput struct {
	Code string `json:"code" jsonschema:"プロジェクトのコード。update_record で空にすると、プロジェクトを外す"`
	Name string `json:"name,omitempty" jsonschema:"プロジェクトの名前。省略すると、プロジェクトの一覧の名前を使う"`
}

func (p *projectInput) toApp() *dayfile.Project {
	if p == nil {
		return nil
	}
	return &dayfile.Project{Code: p.Code, Name: p.Name}
}

type recordInput struct {
	Title   string        `json:"title" jsonschema:"タスク名"`
	Time    *string       `json:"time,omitempty" jsonschema:"時間。9:30-11:30、-12:00 (直前の記録の終わりから)、13:00- (今まで。今日だけ)、30m、1.5h、90 (分) などを、空白かカンマで区切って複数書ける。省略すると、いつもの時間"`
	Project *projectInput `json:"project,omitempty" jsonschema:"プロジェクト。省略すると、タスクの設定か一番新しい記録から写す"`
	Tags    *[]string     `json:"tags,omitempty" jsonschema:"タグ"`
	Note    *string       `json:"note,omitempty" jsonschema:"そのタスクのメモ"`
	Exclude *bool         `json:"exclude,omitempty" jsonschema:"工数に計上しない (日報には載る)"`
}

type addRecordsInput struct {
	Date    string        `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Records []recordInput `json:"records" jsonschema:"足す記録"`
}

type updateRecordInput struct {
	Date    string        `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Title   string        `json:"title" jsonschema:"直す記録のタスク名"`
	Time    *string       `json:"time,omitempty" jsonschema:"時間。今の時間の並びをまるごと置き換える。書き方は add_records と同じ。空文字にすると時間不定"`
	Project *projectInput `json:"project,omitempty" jsonschema:"プロジェクト。code を空にすると外す"`
	Tags    *[]string     `json:"tags,omitempty" jsonschema:"タグ。並びをまるごと置き換える"`
	Note    *string       `json:"note,omitempty" jsonschema:"そのタスクのメモ"`
	Exclude *bool         `json:"exclude,omitempty" jsonschema:"工数に計上しない"`
	Done    *bool         `json:"done,omitempty" jsonschema:"完了。そのタスクを最後に記録した日の記録にだけ付けられる"`
}

type titleInput struct {
	Date  string `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Title string `json:"title" jsonschema:"タスク名"`
}

type skipPlansInput struct {
	Date   string   `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Titles []string `json:"titles" jsonschema:"スキップする予定のタスク名"`
}

type setNotesInput struct {
	Date  string         `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Notes map[string]any `json:"notes" jsonschema:"欄の名前と値の組。例: {\"在宅\": true, \"連絡\": \"なし\"}"`
}

type outputInput struct {
	Date string `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Name string `json:"name,omitempty" jsonschema:"出力か出力セットの名前。省略すると既定の出力"`
}

type exportInput struct {
	Date       string `json:"date,omitempty" jsonschema:"日付 (YYYY-MM-DD)。省略すると今日"`
	Name       string `json:"name,omitempty" jsonschema:"出力セットの名前。省略すると既定の出力"`
	OnConflict string `json:"on_conflict,omitempty" jsonschema:"出力先に同じ名前のファイルがあるときの扱い。overwrite (上書き) か rename (名前の後ろに時刻を付けて別に保存)。省略するか ask なら、何も書かずにエラーを返す"`
}

type projectOutput struct {
	Code  string `json:"code"`
	Name  string `json:"name,omitempty"`
	Until string `json:"until,omitempty" jsonschema:"この日まで使える"`
}

type projectsOutput struct {
	Projects []projectOutput `json:"projects"`
}

type tagsOutput struct {
	Tags []string `json:"tags"`
}

type taskOutput struct {
	Title     string         `json:"title"`
	Project   *projectOutput `json:"project,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Exclude   bool           `json:"exclude,omitempty"`
	UsualTime []string       `json:"usual_time,omitempty" jsonschema:"いつもの時間"`
	LastDate  string         `json:"last_date,omitempty" jsonschema:"最後に記録した日"`
}

type taskListsOutput struct {
	Recent         []taskOutput `json:"recent" jsonschema:"直近のタスク。最後に記録した日が新しい順"`
	Favorites      []taskOutput `json:"favorites" jsonschema:"よく使うタスク"`
	RepeatingToday []taskOutput `json:"repeating_today" jsonschema:"その日に予定が出る定期的なタスク"`
	RepeatingOther []taskOutput `json:"repeating_other" jsonschema:"その日には予定が出ない定期的なタスク"`
	Notices        []string     `json:"notices" jsonschema:"読めずに除いたファイルなどの知らせ"`
}

type recordOutput struct {
	Title   string         `json:"title"`
	Project *projectOutput `json:"project,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
	Note    string         `json:"note,omitempty"`
	Exclude bool           `json:"exclude,omitempty"`
	Done    bool           `json:"done,omitempty"`
	Time    []string       `json:"time" jsonschema:"時間の並び (HH:MM-HH:MM か Nm)。空なら時間不定"`
	Minutes int            `json:"minutes"`
}

type planOutput struct {
	Title     string   `json:"title"`
	UsualTime []string `json:"usual_time,omitempty"`
}

type noteOutput struct {
	Name  string `json:"name"`
	Type  string `json:"type" jsonschema:"text か bool。定義のない欄 (後で消した欄など) は空"`
	Value any    `json:"value"`
}

type projectTotalOutput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Minutes    int    `json:"minutes" jsonschema:"計上する時間の合計 (分)"`
	HasUntimed bool   `json:"has_untimed,omitempty" jsonschema:"計上する記録に時間不定のものがある"`
}

type dayOutput struct {
	Date         string               `json:"date"`
	Exists       bool                 `json:"exists" jsonschema:"その日の日次ファイルがあるか"`
	Records      []recordOutput       `json:"records"`
	Plans        []planOutput         `json:"plans" jsonschema:"まだ記録にしていない予定"`
	Skipped      []string             `json:"skipped" jsonschema:"スキップした予定"`
	Notes        []noteOutput         `json:"notes"`
	Projects     []projectTotalOutput `json:"projects" jsonschema:"プロジェクトごとの合計。記録に出てくる順"`
	TotalMinutes int                  `json:"total_minutes" jsonschema:"計上する時間の合計 (分)"`
	Warnings     []string             `json:"warnings" jsonschema:"期間の外のプロジェクトなどの警告"`
	Notices      []string             `json:"notices" jsonschema:"読めずに除いたファイルなどの知らせ"`
}

type daysOutput struct {
	Days []dayOutput `json:"days"`
}

type writeOutput struct {
	Message  string   `json:"message"`
	Warnings []string `json:"warnings" jsonschema:"期間の外のプロジェクトなどの警告"`
}

type renderedOutput struct {
	Name     string `json:"name"`
	FileName string `json:"file_name"`
	Text     string `json:"text"`
}

type renderOutputResult struct {
	Outputs  []renderedOutput `json:"outputs"`
	Warnings []string         `json:"warnings" jsonschema:"期間の外のプロジェクトなどの警告"`
	Notices  []string         `json:"notices" jsonschema:"読めずに除いたテンプレートなどの知らせ"`
}

type exportResult struct {
	Written  []string `json:"written" jsonschema:"書いたファイルの名前"`
	Warnings []string `json:"warnings" jsonschema:"期間の外のプロジェクトなどの警告"`
	Notices  []string `json:"notices" jsonschema:"読めずに除いたテンプレートなどの知らせ"`
}

func newWriteOutput(warnings []string) writeOutput {
	return writeOutput{Message: "書きました", Warnings: nonNil(warnings)}
}

// nonNil は、空の並びを null ではなく [] で返すためのもの。
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func projectOf(p dayfile.Project) *projectOutput {
	if p.Code == "" {
		return nil
	}
	return &projectOutput{Code: p.Code, Name: p.Name}
}

func timeStrings(es []timeexpr.Entry) []string {
	ss := make([]string, len(es))
	for i, e := range es {
		ss[i] = e.String()
	}
	return ss
}

func itemOutput(it items.Item) taskOutput {
	return taskOutput{
		Title:     it.Title,
		Project:   projectOf(it.Project),
		Tags:      it.Tags,
		Exclude:   it.Exclude,
		UsualTime: timeStrings(it.Time),
	}
}

func itemOutputs(list []items.Item) []taskOutput {
	out := []taskOutput{}
	for _, it := range list {
		out = append(out, itemOutput(it))
	}
	return out
}

func newTaskListsOutput(l app.TaskLists) taskListsOutput {
	out := taskListsOutput{
		Recent:         []taskOutput{},
		Favorites:      itemOutputs(l.Favorites),
		RepeatingToday: itemOutputs(l.RepeatingOn),
		RepeatingOther: itemOutputs(l.RepeatingOff),
		Notices:        nonNil(l.Skipped),
	}
	// 直近のタスクは、add_records で記録されるのと同じ値を返す。AI が見る値と、実際に記録される値をそろえるため。
	for _, r := range l.Recent {
		out.Recent = append(out.Recent, taskOutput{
			Title:     r.Last.Title,
			Project:   projectOf(r.Base.Project),
			Tags:      r.Base.Tags,
			Exclude:   r.Base.Exclude,
			UsualTime: timeStrings(r.Base.Time),
			LastDate:  r.LastDate.Format(yamlfile.DateLayout),
		})
	}
	return out
}

func newDayOutput(v app.DayView) dayOutput {
	out := dayOutput{
		Date:         v.Day.Date.Format(yamlfile.DateLayout),
		Exists:       v.Exists,
		Records:      []recordOutput{},
		Plans:        []planOutput{},
		Skipped:      nonNil(v.Day.Skipped),
		Notes:        []noteOutput{},
		Projects:     []projectTotalOutput{},
		TotalMinutes: v.Summary.TotalMinutes,
		Warnings:     nonNil(v.Warnings),
		Notices:      nonNil(v.Skipped),
	}
	for _, t := range v.Summary.Tasks {
		out.Records = append(out.Records, recordOutput{
			Title:   t.Title,
			Project: projectOf(t.Project),
			Tags:    t.Tags,
			Note:    t.Note,
			Exclude: t.Exclude,
			Done:    t.Done,
			Time:    timeStrings(t.Time),
			Minutes: t.Minutes,
		})
	}
	for _, p := range v.Plans {
		out.Plans = append(out.Plans, planOutput{Title: p.Title, UsualTime: timeStrings(p.Time)})
	}
	for _, n := range v.Notes {
		out.Notes = append(out.Notes, noteOutputOf(n, v.NoteDefs))
	}
	for _, p := range v.Summary.Projects {
		out.Projects = append(out.Projects, projectTotalOf(p))
	}
	return out
}

func noteOutputOf(n dayfile.Note, defs []settings.NoteDef) noteOutput {
	out := noteOutput{Name: n.Name, Value: n.Text}
	if n.Kind == dayfile.NoteBool {
		out.Value = n.Bool
	}
	for _, d := range defs {
		if d.Name == n.Name {
			out.Type = string(d.Type)
		}
	}
	return out
}

func projectTotalOf(p effort.Project) projectTotalOutput {
	return projectTotalOutput{Code: p.Code, Name: p.Name, Minutes: p.Minutes, HasUntimed: p.HasUntimed}
}

func newRenderOutputResult(r app.RenderResult) renderOutputResult {
	out := renderOutputResult{Outputs: []renderedOutput{}, Warnings: nonNil(r.Warnings), Notices: nonNil(r.Skipped)}
	for _, o := range r.Outputs {
		out.Outputs = append(out.Outputs, renderedOutput{Name: o.Name, FileName: o.FileName, Text: o.Text})
	}
	return out
}

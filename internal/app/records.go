package app

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/crepidoma/lumetio/internal/tasklist"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
	"golang.org/x/text/unicode/norm"
)

// NewRecord は、記録に足すタスク 1 つ分の指定。
// nil の項目は指定がないことを表し、タスクの設定や過去の記録から決める。
type NewRecord struct {
	Title string
	// Time は時間の書き方 (時刻や長さ) の文字列。nil ならいつもの時間を使う。
	Time    *string
	Project *dayfile.Project
	Tags    *[]string
	Note    *string
	Exclude *bool
	// Untimed が true なら時間不定で足す。Time に時間を書いたときとは一緒に指定できない。
	Untimed *bool
}

// RecordPatch は、記録を直すときの指定。nil の項目は変えない。
type RecordPatch struct {
	// Title は、その日の記録の名前だけを変える。打ち間違いを直すためのもので、タスクの設定の名前は変えない。
	Title *string
	// Time は時間の書き方の文字列。時間の並びをまるごと置き換える。空文字なら時間を消す。
	// 時間を入れると、時間を決めたことになるので、時間不定は外れる。
	Time *string
	// Untimed は時間不定にするか。時間がある記録は時間不定にできない (Time で同時に消せばできる)。
	Untimed *bool
	// Project のコードが空なら、プロジェクトを外す。
	Project *dayfile.Project
	Tags    *[]string
	Note    *string
	Exclude *bool
	Done    *bool
}

// NoteValue は、メモ欄 1 つに書く値。Value は文字列か真偽値。
type NoteValue struct {
	Name  string
	Value any
	// Current は、画面で見ていたテキストの値。nil でなければ、ファイルの値と違うときに書かずに ErrConflict を返す。
	// 打っているあいだに他の PC が同じ欄を書き換えていたら、古い画面から上書きしないため。
	Current *string
}

// AddRecords は、その日の記録にタスクをまとめて足す。
// 1 つでも誤りがあれば、何も書かない。
// 足し方は画面の＋と同じ。
// 予定と同じタスクを足すと、予定は記録になる。
// その日にもう同じタスクの記録があれば、新しい行は作らず、その記録に時間を足す。
// 時間もいつもの時間もなければ、時間未入力で足す。画面の＋は、足した後に時間を打つため。
func (a *App) AddRecords(date time.Time, records []NewRecord) (warnings []string, err error) {
	return a.addRecords(date, records, false)
}

// AddRecordsExplicit は、AddRecords と同じく記録を足すが、時間か時間不定がはっきり決まらない記録は、何も書かずに入力の誤りにする。MCP から使う。
// AI が時間を入れ忘れた記録を、黙って時間未入力や時間不定で作らないため。
// もう記録があるタスクに、足す時間も時間不定の指定もないときも誤りにする。何も変わらないのに、足したと思われないため。
func (a *App) AddRecordsExplicit(date time.Time, records []NewRecord) (warnings []string, err error) {
	return a.addRecords(date, records, true)
}

func (a *App) addRecords(date time.Time, records []NewRecord, explicit bool) (warnings []string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(records) == 0 {
		return nil, inputErrorf("足す記録がありません")
	}
	s, err := a.loadState()
	if err != nil {
		return nil, err
	}
	day, v, err := a.loadDayForWrite(date, s.settings.Notes)
	if err != nil {
		return nil, err
	}

	for _, rec := range records {
		title := cleanName(rec.Title)
		if title == "" {
			return nil, inputErrorf("タスク名がありません")
		}
		project, err := checkProject(rec.Project, s.settings.Projects)
		if err != nil {
			return nil, err
		}
		untimed := rec.Untimed != nil && *rec.Untimed
		if untimed && rec.Time != nil && strings.TrimSpace(*rec.Time) != "" {
			return nil, inputErrorf("%s: 時間と時間不定は一緒に指定できません", title)
		}

		if i := slices.IndexFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == title }); i >= 0 {
			t := &day.Tasks[i]
			// 時間を足すだけなので、ほかの値を黙って捨てないよう、違う値はエラーにする。
			if differs(rec, project, *t) {
				return nil, inputErrorf("%s はもう記録にあります。プロジェクトなどを変えるときは、記録を直します", title)
			}
			if untimed {
				if err := setUntimed(t); err != nil {
					return nil, err
				}
				continue
			}
			// 時間を書いていなければ、いつもの時間を足す。
			entries := a.usualTime(title, s)
			if rec.Time != nil {
				if entries, err = a.parseTime(*rec.Time, date, day, -1, -1); err != nil {
					return nil, err
				}
			}
			if explicit && len(entries) == 0 {
				return nil, inputErrorf("%s はもう記録にあります。足す時間を指定します", title)
			}
			addTime(t, entries)
			continue
		}

		t, skipped, err := a.baseRecord(title, date, s)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, skipped...)
		if rec.Project != nil {
			t.Project = project
		}
		if rec.Tags != nil {
			t.Tags = cleanTags(*rec.Tags)
		}
		if rec.Note != nil {
			t.Note = nfc(*rec.Note)
		}
		if rec.Exclude != nil {
			t.Exclude = *rec.Exclude
		}
		// 時間を書けば、いつもの時間の代わりにその時間で入り、時間不定は外れる。空文字なら時間は入らず、いつもの時間を時間不定にしたタスクだけが時間不定のまま残る。
		if rec.Time != nil {
			entries, err := a.parseTime(*rec.Time, date, day, -1, -1)
			if err != nil {
				return nil, err
			}
			t.Time = nil
			addTime(&t, entries)
		}
		// 時間不定を指定すれば、いつもの時間があっても時間不定で入る。
		if untimed {
			t.Time = nil
			t.Untimed = true
		}
		if explicit && len(t.Time) == 0 && !t.Untimed {
			return nil, inputErrorf("%s の時間がありません。時間を指定するか、時間不定にするなら untimed に true を指定します", title)
		}
		day.Tasks = append(day.Tasks, t)
	}

	if _, err := a.store.SaveDay(day, v); err != nil {
		return nil, err
	}
	return append(warnings, outOfPeriodWarnings(day, s.settings.Projects)...), nil
}

// UpdateRecord は、その日の記録を直す。
func (a *App) UpdateRecord(date time.Time, title string, patch RecordPatch) (warnings []string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	title = cleanName(title)
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	day, v, i, err := a.loadRecord(date, title)
	if err != nil {
		return nil, err
	}
	t := &day.Tasks[i]

	if patch.Title != nil {
		newTitle := cleanName(*patch.Title)
		if newTitle == "" {
			return nil, inputErrorf("タスク名がありません")
		}
		if newTitle != title && slices.ContainsFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == newTitle }) {
			return nil, inputErrorf("%s はもう記録にあります", newTitle)
		}
		t.Title = newTitle
	}
	if patch.Time != nil {
		entries, err := a.parseTime(*patch.Time, date, day, i, -1)
		if err != nil {
			return nil, err
		}
		t.Time = nil
		addTime(t, entries)
	}
	if patch.Untimed != nil {
		if *patch.Untimed {
			if err := setUntimed(t); err != nil {
				return nil, err
			}
		} else {
			t.Untimed = false
		}
	}
	if patch.Project != nil {
		if t.Project, err = checkProject(patch.Project, st.Projects); err != nil {
			return nil, err
		}
	}
	if patch.Tags != nil {
		t.Tags = cleanTags(*patch.Tags)
	}
	if patch.Note != nil {
		t.Note = nfc(*patch.Note)
	}
	if patch.Exclude != nil {
		t.Exclude = *patch.Exclude
	}
	if patch.Done != nil {
		if *patch.Done {
			later, skipped, err := a.laterRecordDate(t.Title, date)
			if err != nil {
				return nil, err
			}
			warnings = append(warnings, skipped...)
			// 完了の印は、そのタスクの最後の記録にしか効かない。
			if !later.IsZero() {
				return nil, inputErrorf("%s は %s にも記録があります。完了は、最後に記録した日に付けます", t.Title, later.Format(yamlfile.DateLayout))
			}
		}
		t.Done = *patch.Done
	}

	if _, err := a.store.SaveDay(day, v); err != nil {
		return nil, err
	}
	return append(warnings, outOfPeriodWarnings(day, st.Projects)...), nil
}

// ReplaceTime は、記録の時間の index 番目を input で置き換える。
// input は時間の書き方で、複数書けば並びに入り、空なら時間を消す。
// 画面で時間のチップを 1 つだけ直すためのもの。
// current は、画面で見ていたその時間の保存形式。ファイルのその位置の時間と違えば、書かずに ErrConflict を返す。
// 位置だけで指すと、画面が古いまま他で時間が足されたときに、別の時間を直したり消したりしてしまうため。
func (a *App) ReplaceTime(date time.Time, title string, index int, current, input string) (warnings []string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	day, v, i, err := a.loadRecord(date, title)
	if err != nil {
		return nil, err
	}
	t := &day.Tasks[i]
	if index < 0 || index >= len(t.Time) || t.Time[index].String() != current {
		return nil, fmt.Errorf("%s の時間が変わっています: %w", t.Title, ErrConflict)
	}
	entries, err := a.parseTime(input, date, day, i, index)
	if err != nil {
		return nil, err
	}
	t.Time = slices.Concat(t.Time[:index], entries, t.Time[index+1:])
	if _, err := a.store.SaveDay(day, v); err != nil {
		return nil, err
	}
	return outOfPeriodWarnings(day, st.Projects), nil
}

// PreviewTime は、時間の書き方を、保存するときと同じ決まりで解釈して返す。ファイルは書かない。
// 画面で、入力中に解釈した結果を見せるためのもの。
// index が -1 なら title の記録に足すとき (title の記録がなければ新しく足すとき)、0 以上なら index 番目の時間を置き換えるとき。
func (a *App) PreviewTime(date time.Time, title string, index int, input string) ([]timeexpr.Entry, error) {
	day, _, err := a.store.LoadDay(date)
	if err != nil {
		return nil, err
	}
	self := -1
	if index >= 0 {
		title = cleanName(title)
		self = slices.IndexFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == title })
		if self < 0 || index >= len(day.Tasks[self].Time) {
			return nil, inputErrorf("%s の %d 番目の時間はありません", title, index+1)
		}
	}
	return a.parseTime(input, date, day, self, index)
}

// RemoveRecord は、その日の記録を消す。
func (a *App) RemoveRecord(date time.Time, title string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	day, v, i, err := a.loadRecord(date, title)
	if err != nil {
		return err
	}
	day.Tasks = slices.Delete(day.Tasks, i, i+1)
	_, err = a.store.SaveDay(day, v)
	return err
}

// SkipPlans は、その日の予定をまとめてスキップする。
func (a *App) SkipPlans(date time.Time, titles []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(titles) == 0 {
		return inputErrorf("スキップする予定がありません")
	}
	s, err := a.loadState()
	if err != nil {
		return err
	}
	day, v, err := a.loadDayForWrite(date, s.settings.Notes)
	if err != nil {
		return err
	}
	plans := tasklist.Plans(day, s.items)
	for _, title := range titles {
		title = cleanName(title)
		// もうスキップした予定は、もう一度スキップしても変わらない。
		if slices.Contains(day.Skipped, title) {
			continue
		}
		if !slices.ContainsFunc(plans, func(p items.Item) bool { return p.Title == title }) {
			return inputErrorf("%s はその日の予定にありません", title)
		}
		day.Skipped = append(day.Skipped, title)
	}
	_, err = a.store.SaveDay(day, v)
	return err
}

// SetNotes は、その日のメモ欄をまとめて書く。
// 書けるのは settings.yaml に定義のある欄だけ。
// 書き間違えた名前で、その日にだけある欄を作らないため。
func (a *App) SetNotes(date time.Time, values []NoteValue) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(values) == 0 {
		return inputErrorf("書くメモ欄がありません")
	}
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return err
	}
	day, v, err := a.loadDayForWrite(date, st.Notes)
	if err != nil {
		return err
	}
	for _, nv := range values {
		nv.Name = cleanName(nv.Name)
		if str, ok := nv.Value.(string); ok {
			nv.Value = nfc(str)
		}
		i := slices.IndexFunc(st.Notes, func(d settings.NoteDef) bool { return d.Name == nv.Name })
		if i < 0 {
			return inputErrorf("メモ欄 %s はありません。書ける欄: %s", nv.Name, noteDefsText(st.Notes))
		}
		n, err := noteFor(st.Notes[i], nv.Value)
		if err != nil {
			return err
		}
		j := slices.IndexFunc(day.Notes, func(x dayfile.Note) bool { return x.Name == n.Name })
		if nv.Current != nil {
			cur := ""
			if j >= 0 {
				cur = day.Notes[j].Text
			}
			if cur != nfc(*nv.Current) {
				return fmt.Errorf("メモ欄 %s が変わっています: %w", n.Name, ErrConflict)
			}
		}
		if j >= 0 {
			day.Notes[j] = n
		} else {
			day.Notes = append(day.Notes, n)
		}
	}
	_, err = a.store.SaveDay(day, v)
	return err
}

func noteFor(def settings.NoteDef, value any) (dayfile.Note, error) {
	switch def.Type {
	case settings.NoteBool:
		b, ok := value.(bool)
		if !ok {
			return dayfile.Note{}, inputErrorf("メモ欄 %s は真偽値 (true か false) で書きます", def.Name)
		}
		return dayfile.Note{Name: def.Name, Kind: dayfile.NoteBool, Bool: b}, nil
	default:
		s, ok := value.(string)
		if !ok {
			return dayfile.Note{}, inputErrorf("メモ欄 %s は文字列で書きます", def.Name)
		}
		return dayfile.Note{Name: def.Name, Kind: dayfile.NoteText, Text: s}, nil
	}
}

func noteDefsText(defs []settings.NoteDef) string {
	if len(defs) == 0 {
		return "なし"
	}
	var parts []string
	for _, d := range defs {
		t := "テキスト"
		if d.Type == settings.NoteBool {
			t = "真偽値"
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", d.Name, t))
	}
	return strings.Join(parts, "、")
}

// loadRecord は、書き換えるためにその日を読み、タスクの記録の位置を返す。
func (a *App) loadRecord(date time.Time, title string) (dayfile.Day, store.Version, int, error) {
	title = cleanName(title)
	day, v, err := a.store.LoadDay(date)
	if err != nil {
		return dayfile.Day{}, store.Version{}, 0, err
	}
	i := slices.IndexFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == title })
	if i < 0 {
		return dayfile.Day{}, store.Version{}, 0, inputErrorf("%s に %s の記録はありません", yamlfile.DateOnly(date).Format(yamlfile.DateLayout), title)
	}
	return day, v, i, nil
}

// baseRecord は、まだその日にないタスクを足すときの記録を、画面の＋と同じ決まりで作る。
// タスクの設定があればその値だけを使い、なければその日までで一番新しい記録からプロジェクト、タグ、計上しないを写す。
// どちらもなければ、新しいタスクとして空から作る。
func (a *App) baseRecord(title string, date time.Time, s state) (dayfile.Task, []string, error) {
	if it, ok := tasklist.Find(s.items, title); ok {
		return tasklist.FromItem(it), nil, nil
	}
	last, _, found, skipped, err := a.lastRecord(title, date)
	if err != nil {
		return dayfile.Task{}, nil, err
	}
	if !found {
		return dayfile.Task{Title: title}, skipped, nil
	}
	return tasklist.FromItem(tasklist.NewItem(last)), skipped, nil
}

// lastRecord は、date より前で、そのタスクを一番最後に記録した記録と、その日を探す。
// 記録がなければ、found は false。
// 読めないファイルは飛ばして、skipped に理由を返す。
func (a *App) lastRecord(title string, date time.Time) (last dayfile.Task, on time.Time, found bool, skipped []string, err error) {
	limit := yamlfile.DateOnly(date)
	skipped, err = a.eachDayNewestFirst(func(d time.Time) bool { return d.Before(limit) }, func(d time.Time, day dayfile.Day) bool {
		if j := slices.IndexFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == title }); j >= 0 {
			last, on, found = day.Tasks[j], d, true
			return true
		}
		return false
	})
	return last, on, found, skipped, err
}

// laterRecordDate は、date より後でそのタスクを最後に記録した日を返す。なければゼロ値。
func (a *App) laterRecordDate(title string, date time.Time) (later time.Time, skipped []string, err error) {
	limit := yamlfile.DateOnly(date)
	skipped, err = a.eachDayNewestFirst(func(d time.Time) bool { return d.After(limit) }, func(d time.Time, day dayfile.Day) bool {
		if slices.ContainsFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == title }) {
			later = d
			return true
		}
		return false
	})
	return later, skipped, err
}

// eachDayNewestFirst は、日次ファイルを新しい日から順に読み、fn に渡す。fn が true を返したら止める。
// use が false の日は読まずに飛ばす。読めないファイルは除いて、skipped に理由を返す。
// 1 つのファイルの誤りで、ほかの日の記録を探せなくならないようにするため。
func (a *App) eachDayNewestFirst(use func(time.Time) bool, fn func(time.Time, dayfile.Day) bool) (skipped []string, err error) {
	dates, err := a.store.DayDates()
	if err != nil {
		return nil, err
	}
	for i := len(dates) - 1; i >= 0; i-- {
		if !use(dates[i]) {
			continue
		}
		day, _, err := a.store.LoadDay(dates[i])
		if err != nil {
			skipped = append(skipped, err.Error())
			continue
		}
		if fn(dates[i], day) {
			break
		}
	}
	return skipped, nil
}

// usualTime は、タスクの設定にあるいつもの時間を返す。
func (a *App) usualTime(title string, s state) []timeexpr.Entry {
	if it, ok := tasklist.Find(s.items, title); ok {
		return slices.Clone(it.Time)
	}
	return nil
}

// parseTime は時間の書き方を解釈する。
// self は書き換える記録の位置、selfEntry はその中で書き換える時間の位置。
// 書き換えて消える時間は「直前の記録」の候補に入れない。
// selfEntry が -1 なら記録の時間をすべて置き換え、self も -1 なら足すだけ。
func (a *App) parseTime(input string, date time.Time, day dayfile.Day, self, selfEntry int) ([]timeexpr.Entry, error) {
	var ctx timeexpr.Context
	for i, t := range day.Tasks {
		for j, e := range t.Time {
			if i == self && (selfEntry < 0 || j == selfEntry) {
				continue
			}
			if e.IsRange {
				ctx.Ranges = append(ctx.Ranges, e)
			}
		}
	}
	// 「今まで」は今日だけで使える。
	if a.isToday(date) {
		now := a.minuteOfDay()
		ctx.Now = &now
	}
	return timeexpr.Parse(input, ctx)
}

// checkProject は、指定されたプロジェクトを確かめる。
// 名前が書かれていなければ、プロジェクトの一覧の名前を使う。
func checkProject(p *dayfile.Project, list []settings.Project) (dayfile.Project, error) {
	if p == nil {
		return dayfile.Project{}, nil
	}
	code := cleanName(p.Code)
	name := cleanName(p.Name)
	if code == "" {
		if name != "" {
			return dayfile.Project{}, inputErrorf("プロジェクト %s にコードがありません", name)
		}
		return dayfile.Project{}, nil
	}
	if name == "" {
		for _, sp := range list {
			if sp.Code == code {
				name = sp.Name
			}
		}
	}
	return dayfile.Project{Code: code, Name: name}, nil
}

// differs は、すでにある記録と違う値が指定されているか。
// プロジェクトはコードで比べる。名前は、指定したときだけ比べる。
// 名前を省くと一覧の今の名前で埋まるので、そのまま比べると、一覧で名前を変えただけで同じプロジェクトが違うものになるため。
func differs(rec NewRecord, project dayfile.Project, t dayfile.Task) bool {
	if rec.Project != nil {
		if project.Code != t.Project.Code {
			return true
		}
		if cleanName(rec.Project.Name) != "" && project.Name != t.Project.Name {
			return true
		}
	}
	if rec.Tags != nil && !slices.Equal(cleanTags(*rec.Tags), t.Tags) {
		return true
	}
	if rec.Note != nil && nfc(*rec.Note) != t.Note {
		return true
	}
	return rec.Exclude != nil && *rec.Exclude != t.Exclude
}

func cleanTags(tags []string) []string {
	var out []string
	for _, tag := range tags {
		tag = cleanName(tag)
		if tag != "" && !slices.Contains(out, tag) {
			out = append(out, tag)
		}
	}
	return out
}

// cleanName は、名前 (タスク名、コード、欄や出力の名前など) を、前後の空白を除いて NFC にそろえる。
// 入口ごとにそろえ方がずれると、同じ名前を違うものとして扱ってしまうため、受け取った名前はすべてこれを通す。
func cleanName(s string) string {
	return nfc(strings.TrimSpace(s))
}

// nfc は、受け取った文字列を NFC にそろえる。
// ファイルから読んだ値は NFC なので、macOS などから渡された名前でも同じものとして比べるため。
func nfc(s string) string {
	return norm.NFC.String(s)
}

// addTime は、記録に時間を足す。時間を足すと、時間を決めたことになるので、時間不定は外す。
func addTime(t *dayfile.Task, entries []timeexpr.Entry) {
	if len(entries) == 0 {
		return
	}
	t.Time = append(t.Time, entries...)
	t.Untimed = false
}

// setUntimed は、記録を時間不定にする。時間と時間不定の両方を持たないよう、時間がある記録は入力の誤りにする。
func setUntimed(t *dayfile.Task) error {
	if len(t.Time) > 0 {
		return inputErrorf("%s には時間があるので、時間不定にできません。時間を消すと時間不定にできます", t.Title)
	}
	t.Untimed = true
	return nil
}

// missingTimeWarnings は、計上する記録 (計上しないでないもの) の時間未入力を知らせる文を返す。
// 時間の入れ忘れに、工数を取り込む前に気づけるようにするため。時間不定と計上しない記録は、工数の取り込みに関わらないので知らせない。
func missingTimeWarnings(day dayfile.Day) []string {
	var warnings []string
	for _, t := range day.Tasks {
		if !t.Exclude && !t.Untimed && len(t.Time) == 0 {
			warnings = append(warnings, fmt.Sprintf("%s の時間が入っていません", t.Title))
		}
	}
	return warnings
}

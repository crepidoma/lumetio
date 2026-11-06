// Package items は items.yaml の中身を読み書きする。
package items

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
	"go.yaml.in/yaml/v3"
)

// Item は 1 つのタスクの設定。
// ＋で足すときや予定を実施済にするときは、ここに書いたプロジェクト、タグ、計上しないだけを使い、過去の記録で穴を埋めない。
type Item struct {
	Title string
	// Project の Code が空なら、プロジェクトを設定していない。
	Project dayfile.Project
	Tags    []string
	Exclude bool
	// Time はいつもの時間。
	Time []timeexpr.Entry
	// Untimed は、いつもの時間を時間不定にすること。Time とは一緒に持たない。
	Untimed bool
	// Repeat が nil なら定期的なタスクではない。
	Repeat   *Repeat
	Favorite bool
}

// HasSettings は、タスクに設定 (よく使う、いつもの時間、繰り返し) があるかを返す。時間不定は、いつもの時間の設定の 1 つ。
// プロジェクト、タグ、計上しないは、設定を付けたタスクを記録するときに使う値で、それだけでは設定ではない。
// 設定をすべて外したタスクは、items.yaml に残さない。
func (it Item) HasSettings() bool {
	return len(it.Time) > 0 || it.Untimed || it.Repeat != nil || it.Favorite
}

// Repeat はタスクの繰り返し。
// 語と意味は iCalendar (RFC 5545) の RRULE に合わせ、画面で作れる組み合わせだけを受け付ける。
// 画面で開けない設定ができないようにするため。
type Repeat struct {
	Freq Freq
	// Interval は 1 か 2。2 は隔週で、weekly のときだけ使える。
	Interval int
	ByDay    []DaySpec
	// ByMonthDay は毎月の日 (1〜31)。0 なら使わない。
	ByMonthDay int
	// Start はいつから。隔週では数え始めも兼ねる。ゼロ値なら指定なし。
	Start time.Time
	// Until はいつまで (この日を含む)。ゼロ値なら指定なし。
	Until time.Time
}

// Freq は繰り返しの頻度。
type Freq string

const (
	Daily   Freq = "daily"
	Weekly  Freq = "weekly"
	Monthly Freq = "monthly"
)

// DaySpec は by_day の 1 つ分。
type DaySpec struct {
	// N は monthly での週の番号 (1〜5、-1 は最終)。weekly では 0。
	N       int
	Weekday time.Weekday
}

var weekdayNames = []string{"su", "mo", "tu", "we", "th", "fr", "sa"}

var daySpecPattern = regexp.MustCompile(`^(-1|[1-5])?(su|mo|tu|we|th|fr|sa)$`)

func (d DaySpec) String() string {
	if d.N == 0 {
		return weekdayNames[d.Weekday]
	}
	return strconv.Itoa(d.N) + weekdayNames[d.Weekday]
}

// ParseDaySpec は by_day の 1 つ分 (mo、2mo、-1fr など) を読む。
func ParseDaySpec(s string) (DaySpec, bool) {
	m := daySpecPattern.FindStringSubmatch(s)
	if m == nil {
		return DaySpec{}, false
	}
	var d DaySpec
	if m[1] != "" {
		d.N, _ = strconv.Atoi(m[1])
	}
	for i, name := range weekdayNames {
		if name == m[2] {
			d.Weekday = time.Weekday(i)
		}
	}
	return d, true
}

type rawItem struct {
	Title    string               `yaml:"title"`
	Project  *dayfile.ProjectYAML `yaml:"project"`
	Tags     []string             `yaml:"tags"`
	Exclude  bool                 `yaml:"exclude"`
	Time     []string             `yaml:"time"`
	Untimed  bool                 `yaml:"untimed"`
	Repeat   *rawRepeat           `yaml:"repeat"`
	Favorite bool                 `yaml:"favorite"`
}

type rawRepeat struct {
	Freq       string   `yaml:"freq"`
	Interval   int      `yaml:"interval"`
	ByDay      []string `yaml:"by_day"`
	ByMonthDay []int    `yaml:"by_month_day"`
	Start      string   `yaml:"start"`
	Until      string   `yaml:"until"`
}

// Parse は items.yaml の中身を読む。
func Parse(data []byte) ([]Item, error) {
	var raw []rawItem
	if err := yamlfile.Decode(data, &raw); err != nil {
		return nil, fmt.Errorf("items.yaml を読めません: %w", err)
	}

	titles := map[string]bool{}
	var items []Item
	for i, ri := range raw {
		if strings.TrimSpace(ri.Title) == "" {
			return nil, fmt.Errorf("items.yaml の %d 件目に title がありません", i+1)
		}
		// タスクはタイトルで見分けるので、重なるとどちらの設定を使うか決められない。
		if titles[ri.Title] {
			return nil, fmt.Errorf("items.yaml でタスク %q が重なっています", ri.Title)
		}
		titles[ri.Title] = true

		it := Item{Title: ri.Title, Tags: ri.Tags, Exclude: ri.Exclude, Untimed: ri.Untimed, Favorite: ri.Favorite}
		var err error
		if it.Project, err = ri.Project.Project(); err != nil {
			return nil, fmt.Errorf("タスク %q: %w", ri.Title, err)
		}
		if it.Time, err = dayfile.ParseTimes(ri.Time); err != nil {
			return nil, fmt.Errorf("タスク %q: %w", ri.Title, err)
		}
		// いつもの時間があるのか時間不定なのか決まらないので、両方は持たない。
		if it.Untimed && len(it.Time) > 0 {
			return nil, fmt.Errorf("タスク %q: time と untimed は一緒に書けません", ri.Title)
		}
		if ri.Repeat != nil {
			r, err := parseRepeat(ri.Repeat)
			if err != nil {
				return nil, fmt.Errorf("タスク %q の repeat: %w", ri.Title, err)
			}
			it.Repeat = &r
		}
		items = append(items, it)
	}
	return items, nil
}

func parseRepeat(raw *rawRepeat) (Repeat, error) {
	r := Repeat{Freq: Freq(raw.Freq), Interval: raw.Interval}
	if r.Interval == 0 {
		r.Interval = 1
	}
	for _, s := range raw.ByDay {
		d, ok := ParseDaySpec(s)
		if !ok {
			return Repeat{}, fmt.Errorf("by_day の %q を読めません", s)
		}
		r.ByDay = append(r.ByDay, d)
	}
	var err error
	if raw.Start != "" {
		if r.Start, err = time.Parse(yamlfile.DateLayout, raw.Start); err != nil {
			return Repeat{}, fmt.Errorf("start %q を読めません", raw.Start)
		}
	}
	if raw.Until != "" {
		if r.Until, err = time.Parse(yamlfile.DateLayout, raw.Until); err != nil {
			return Repeat{}, fmt.Errorf("until %q を読めません", raw.Until)
		}
	}
	if !r.Start.IsZero() && !r.Until.IsZero() && r.Until.Before(r.Start) {
		return Repeat{}, fmt.Errorf("until が start より前です")
	}

	switch r.Freq {
	case Daily:
		if len(r.ByDay) > 0 || len(raw.ByMonthDay) > 0 {
			return Repeat{}, fmt.Errorf("daily には by_day も by_month_day も書けません")
		}
	case Weekly:
		if len(r.ByDay) == 0 {
			return Repeat{}, fmt.Errorf("weekly には by_day が必要です")
		}
		if len(raw.ByMonthDay) > 0 {
			return Repeat{}, fmt.Errorf("weekly には by_month_day を書けません")
		}
		seen := map[time.Weekday]bool{}
		for _, d := range r.ByDay {
			if d.N != 0 {
				return Repeat{}, fmt.Errorf("weekly の by_day に週の番号 (%s) は付けられません", d)
			}
			if seen[d.Weekday] {
				return Repeat{}, fmt.Errorf("by_day の %s が重なっています", d)
			}
			seen[d.Weekday] = true
		}
	case Monthly:
		switch {
		case len(r.ByDay) > 0 && len(raw.ByMonthDay) > 0:
			return Repeat{}, fmt.Errorf("monthly には by_day と by_month_day のどちらか 1 つを書きます")
		case len(r.ByDay) == 1:
			if r.ByDay[0].N == 0 {
				return Repeat{}, fmt.Errorf("monthly の by_day には週の番号を付けます (例: 2mo)")
			}
		case len(raw.ByMonthDay) == 1:
			if d := raw.ByMonthDay[0]; d < 1 || d > 31 {
				return Repeat{}, fmt.Errorf("by_month_day の %d は 1〜31 で書きます", d)
			}
			r.ByMonthDay = raw.ByMonthDay[0]
		default:
			return Repeat{}, fmt.Errorf("monthly には by_day か by_month_day を 1 つだけ書きます")
		}
	default:
		return Repeat{}, fmt.Errorf("freq %q は使えません。daily、weekly、monthly のどれかを書きます", raw.Freq)
	}

	switch r.Interval {
	case 1:
	case 2:
		if r.Freq != Weekly {
			return Repeat{}, fmt.Errorf("interval: 2 は weekly でだけ使えます")
		}
		// 隔週は、どの週から数えるかが決まらないと出す日が決まらない。
		if r.Start.IsZero() {
			return Repeat{}, fmt.Errorf("隔週には start が必要です")
		}
	default:
		return Repeat{}, fmt.Errorf("interval は 1 か 2 で書きます")
	}
	return r, nil
}

// Marshal は items.yaml の中身を書く。
// 設定のないタスクは書かない。
func Marshal(items []Item) ([]byte, error) {
	root := yamlfile.Sequence()
	for _, it := range items {
		if !it.HasSettings() {
			continue
		}
		// いつもの時間と時間不定の両方を持つタスクは、読むときにエラーになる。書いてから開けなくならないよう、書く前に止める。
		if it.Untimed && len(it.Time) > 0 {
			return nil, fmt.Errorf("タスク %q はいつもの時間と時間不定の両方を持っています", it.Title)
		}
		root.Content = append(root.Content, itemNode(it))
	}

	// タスクがないときに "[]" と書かないよう、空のファイルにする。
	if len(root.Content) == 0 {
		return nil, nil
	}
	data, err := yamlfile.Encode(root)
	if err != nil {
		return nil, fmt.Errorf("items.yaml を書けません: %w", err)
	}
	return data, nil
}

func itemNode(it Item) *yaml.Node {
	n := yamlfile.Mapping()
	yamlfile.Add(n, "title", yamlfile.Text(it.Title))
	if p := dayfile.ProjectNode(it.Project); p != nil {
		yamlfile.Add(n, "project", p)
	}
	if len(it.Tags) > 0 {
		yamlfile.Add(n, "tags", yamlfile.FlowList(it.Tags))
	}
	if it.Exclude {
		yamlfile.Add(n, "exclude", yamlfile.Bool(true))
	}
	if len(it.Time) > 0 {
		yamlfile.Add(n, "time", dayfile.TimesNode(it.Time))
	}
	if it.Untimed {
		yamlfile.Add(n, "untimed", yamlfile.Bool(true))
	}
	if it.Repeat != nil {
		yamlfile.Add(n, "repeat", repeatNode(*it.Repeat))
	}
	if it.Favorite {
		yamlfile.Add(n, "favorite", yamlfile.Bool(true))
	}
	return n
}

func repeatNode(r Repeat) *yaml.Node {
	n := yamlfile.Mapping()
	yamlfile.Add(n, "freq", yamlfile.Text(string(r.Freq)))
	if r.Interval > 1 {
		yamlfile.Add(n, "interval", yamlfile.Int(r.Interval))
	}
	if len(r.ByDay) > 0 {
		days := make([]string, len(r.ByDay))
		for i, d := range r.ByDay {
			days[i] = d.String()
		}
		yamlfile.Add(n, "by_day", yamlfile.FlowList(days))
	}
	if r.ByMonthDay != 0 {
		list := yamlfile.FlowList(nil)
		list.Content = append(list.Content, yamlfile.Int(r.ByMonthDay))
		yamlfile.Add(n, "by_month_day", list)
	}
	if !r.Start.IsZero() {
		yamlfile.Add(n, "start", yamlfile.Date(r.Start))
	}
	if !r.Until.IsZero() {
		yamlfile.Add(n, "until", yamlfile.Date(r.Until))
	}
	return n
}

// Package timeexpr は、タスクの時間の書き方を解釈する。
package timeexpr

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const minutesPerDay = 24 * 60

// Entry は時間の 1 件。開始-終了か、長さのどちらか。
type Entry struct {
	IsRange bool
	// Start と End は 0:00 からの分 (0〜1439)。IsRange のときだけ使う。
	Start, End int
	// Minutes は長さ。開始-終了でも計算済みの値を持つ。
	Minutes int
}

// String は保存形式 (`HH:MM-HH:MM` か `Nm`) を返す。
func (e Entry) String() string {
	if e.IsRange {
		return clock(e.Start) + "-" + clock(e.End)
	}
	return strconv.Itoa(e.Minutes) + "m"
}

func clock(m int) string {
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

func newRange(start, end int) Entry {
	return Entry{IsRange: true, Start: start, End: end, Minutes: (end - start + minutesPerDay) % minutesPerDay}
}

// Context は、入力の外にある情報。
type Context struct {
	// Ranges は、その日の全タスクの記録 (予定は含まない) の開始-終了。`-H:MM` の始まりを探すのに使う。
	Ranges []Entry
	// Now は今の時刻 (0:00 からの分)。今日以外の日は nil にする。
	Now *int
	// Detached は、どの日にも結び付かない入力 (いつもの時間など) であること。
	// 「直前の記録の終わりから」と「今まで」は、その日の記録と今の時刻で決まるので、受け付けない。
	Detached bool
}

// InputError は入力の誤り。内部の失敗とは区別して、画面にそのまま出す。
type InputError struct {
	Token  string
	Reason string
}

func (e *InputError) Error() string {
	return fmt.Sprintf("%s: %s", e.Token, e.Reason)
}

// 全角の英数字と記号、日本語の区切りや波ダッシュを半角に寄せる。
// IME によって「～」(U+FF5E) と「〜」(U+301C) のどちらも出るので両方受ける。
func normalize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '！' && r <= '～':
			r -= 0xFEE0
		case r == '　':
			return ' '
		case r == '、':
			return ','
		case r == '〜':
			return '-'
		}
		if r == '~' {
			return '-'
		}
		return r
	}, s)
}

var (
	// 後ろに空白があるハイフンだけ前後の空白を詰める。
	// 「16:30 -17:00」の「-17:00」は直前の終わりからの書き方なので、別の時間として残す。
	spacedHyphen = regexp.MustCompile(`\s*-(\s+|$)`)
	separators   = regexp.MustCompile(`[\s,]+`)

	colonTime   = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
	digitsTime  = regexp.MustCompile(`^\d{1,4}$`)
	integer     = regexp.MustCompile(`^\d+$`)
	decimal     = regexp.MustCompile(`^\d*\.\d+$`)
	unitPattern = regexp.MustCompile(`^(?:(\d*\.?\d+)h)?(?:(\d+)m)?$`)
)

// Parse は入力欄の文字列を解釈する。先に書いた時間から順に返す。
func Parse(input string, ctx Context) ([]Entry, error) {
	s := strings.ToLower(normalize(input))
	s = spacedHyphen.ReplaceAllString(s, "-")
	var entries []Entry
	for _, token := range separators.Split(strings.TrimSpace(s), -1) {
		if token == "" {
			continue
		}
		e, err := parseToken(token, ctx, entries)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func parseToken(token string, ctx Context, earlier []Entry) (Entry, error) {
	if strings.Contains(token, "-") {
		return parseRange(token, ctx, earlier)
	}
	minutes, err := parseDuration(token)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Minutes: minutes}, nil
}

func parseRange(token string, ctx Context, earlier []Entry) (Entry, error) {
	left, right, _ := strings.Cut(token, "-")
	if strings.Contains(right, "-") || (left == "" && right == "") {
		return Entry{}, &InputError{token, "書き方が分かりません"}
	}

	if ctx.Detached && (left == "" || right == "") {
		return Entry{}, &InputError{token, "開始と終了の両方を書きます"}
	}

	var start, end int
	var err error
	switch {
	case left == "":
		if end, err = parseClock(token, right); err != nil {
			return Entry{}, err
		}
		if start, err = previousEnd(token, end, ctx.Ranges, earlier); err != nil {
			return Entry{}, err
		}
	case right == "":
		if start, err = parseClock(token, left); err != nil {
			return Entry{}, err
		}
		if ctx.Now == nil {
			return Entry{}, &InputError{token, "「今まで」は今日だけで使えます"}
		}
		if *ctx.Now <= start {
			return Entry{}, &InputError{token, "まだその時刻になっていません"}
		}
		end = *ctx.Now
	default:
		if start, err = parseClock(token, left); err != nil {
			return Entry{}, err
		}
		if end, err = parseClock(token, right); err != nil {
			return Entry{}, err
		}
	}
	if start == end {
		return Entry{}, &InputError{token, "開始と終了が同じです"}
	}
	return newRange(start, end), nil
}

// parseClock は時刻を 0:00 からの分にする。24:00〜47:59 は翌日の時刻として 24 時間を引く。
func parseClock(token, s string) (int, error) {
	var h, m int
	if g := colonTime.FindStringSubmatch(s); g != nil {
		h, _ = strconv.Atoi(g[1])
		m, _ = strconv.Atoi(g[2])
	} else if digitsTime.MatchString(s) {
		n, _ := strconv.Atoi(s)
		if len(s) <= 2 {
			h = n
		} else {
			h, m = n/100, n%100
		}
	} else {
		return 0, &InputError{token, fmt.Sprintf("%q は時刻として読めません", s)}
	}
	if h >= 48 || m >= 60 {
		return 0, &InputError{token, fmt.Sprintf("%q は時刻として読めません", s)}
	}
	return (h*60 + m) % minutesPerDay, nil
}

// previousEnd は、end 以前に終わっている記録のうち一番遅い終わりを返す。
func previousEnd(token string, end int, ranges, earlier []Entry) (int, error) {
	best := -1
	for _, list := range [][]Entry{ranges, earlier} {
		for _, r := range list {
			if r.IsRange && r.End <= end && r.End > best {
				best = r.End
			}
		}
	}
	if best < 0 {
		return 0, &InputError{token, "それより前に終わった記録がありません"}
	}
	return best, nil
}

func parseDuration(token string) (int, error) {
	var minutes int
	switch {
	case integer.MatchString(token):
		n, err := strconv.Atoi(token)
		if err != nil {
			return 0, &InputError{token, "長さが長すぎます"}
		}
		minutes = n
	case decimal.MatchString(token):
		minutes = hoursToMinutes(token)
	case colonTime.MatchString(token):
		g := colonTime.FindStringSubmatch(token)
		h, _ := strconv.Atoi(g[1])
		m, _ := strconv.Atoi(g[2])
		if m >= 60 {
			return 0, &InputError{token, "分は 59 までです"}
		}
		minutes = h*60 + m
	default:
		g := unitPattern.FindStringSubmatch(token)
		if g == nil || (g[1] == "" && g[2] == "") {
			return 0, &InputError{token, "書き方が分かりません"}
		}
		if g[1] != "" {
			minutes = hoursToMinutes(g[1])
		}
		if g[2] != "" {
			m, err := strconv.Atoi(g[2])
			if err != nil {
				return 0, &InputError{token, "長さが長すぎます"}
			}
			minutes += m
		}
	}
	if minutes == 0 {
		return 0, &InputError{token, "0 分です"}
	}
	if minutes > minutesPerDay {
		return 0, &InputError{token, "24 時間を超えています"}
	}
	return minutes, nil
}

// hoursToMinutes は小数の時間を分にする。分未満は四捨五入する。
// 呼び出し側で数字と小数点だけの文字列に絞っているので、ParseFloat は失敗しない。
func hoursToMinutes(s string) int {
	h, _ := strconv.ParseFloat(s, 64)
	return int(math.Round(h * 60))
}

// ParseStored は日次ファイルに保存された時間 (`HH:MM-HH:MM` か `Nm`) を読む。
func ParseStored(s string) (Entry, error) {
	if left, right, ok := strings.Cut(s, "-"); ok {
		start, err1 := parseStoredClock(left)
		end, err2 := parseStoredClock(right)
		if err1 != nil || err2 != nil || start == end {
			return Entry{}, fmt.Errorf("保存された時間 %q を読めません", s)
		}
		return newRange(start, end), nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
	if err != nil || !strings.HasSuffix(s, "m") || n <= 0 || n > minutesPerDay {
		return Entry{}, fmt.Errorf("保存された時間 %q を読めません", s)
	}
	return Entry{Minutes: n}, nil
}

func parseStoredClock(s string) (int, error) {
	g := colonTime.FindStringSubmatch(s)
	if g == nil {
		return 0, fmt.Errorf("時刻 %q を読めません", s)
	}
	h, _ := strconv.Atoi(g[1])
	m, _ := strconv.Atoi(g[2])
	if h >= 24 || m >= 60 {
		return 0, fmt.Errorf("時刻 %q を読めません", s)
	}
	return h*60 + m, nil
}

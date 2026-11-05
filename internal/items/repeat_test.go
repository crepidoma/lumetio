package items

import (
	"testing"
	"time"
)

// 2026 年 11 月は日曜始まりで、11/2 が月曜、11/30 が月曜。
func TestOccurs(t *testing.T) {
	tests := []struct {
		name   string
		repeat Repeat
		yes    []time.Time
		no     []time.Time
	}{
		{
			name:   "毎日",
			repeat: Repeat{Freq: Daily, Interval: 1},
			yes:    []time.Time{date(2026, 11, 1), date(2026, 11, 7)},
		},
		{
			name:   "平日",
			repeat: Repeat{Freq: Weekly, Interval: 1, ByDay: weekdays(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)},
			yes:    []time.Time{date(2026, 11, 2), date(2026, 11, 6)},
			no:     []time.Time{date(2026, 11, 1), date(2026, 11, 7)},
		},
		{
			name:   "毎週 木",
			repeat: Repeat{Freq: Weekly, Interval: 1, ByDay: weekdays(time.Thursday)},
			yes:    []time.Time{date(2026, 11, 5), date(2026, 11, 12)},
			no:     []time.Time{date(2026, 11, 4), date(2026, 11, 6)},
		},
		{
			// 11/4 (水) を含む週から 1 週おき。週は月曜から始まる。
			name:   "隔週 水",
			repeat: Repeat{Freq: Weekly, Interval: 2, ByDay: weekdays(time.Wednesday), Start: date(2026, 11, 4)},
			yes:    []time.Time{date(2026, 11, 4), date(2026, 11, 18), date(2026, 12, 2)},
			no:     []time.Time{date(2026, 10, 21), date(2026, 11, 11), date(2026, 11, 25)},
		},
		{
			// start が週の途中 (金曜) でも、その週の月曜から数える。
			name:   "隔週 月・金、start が金曜",
			repeat: Repeat{Freq: Weekly, Interval: 2, ByDay: weekdays(time.Monday, time.Friday), Start: date(2026, 11, 6)},
			yes:    []time.Time{date(2026, 11, 6), date(2026, 11, 16), date(2026, 11, 20)},
			// 11/2 (月) は数え始めの週だが、start より前なので出さない。
			no: []time.Time{date(2026, 11, 2), date(2026, 11, 9), date(2026, 11, 13)},
		},
		{
			name:   "毎月 第 2 月曜",
			repeat: Repeat{Freq: Monthly, Interval: 1, ByDay: []DaySpec{{N: 2, Weekday: time.Monday}}},
			yes:    []time.Time{date(2026, 11, 9), date(2026, 12, 14)},
			no:     []time.Time{date(2026, 11, 2), date(2026, 11, 16)},
		},
		{
			name:   "毎月 第 5 月曜",
			repeat: Repeat{Freq: Monthly, Interval: 1, ByDay: []DaySpec{{N: 5, Weekday: time.Monday}}},
			yes:    []time.Time{date(2026, 11, 30)},
			// 2026 年 12 月の月曜は 4 回だけなので、その月には出さない。
			no: []time.Time{date(2026, 12, 28), date(2026, 11, 23)},
		},
		{
			name:   "毎月 最終月曜",
			repeat: Repeat{Freq: Monthly, Interval: 1, ByDay: []DaySpec{{N: -1, Weekday: time.Monday}}},
			yes:    []time.Time{date(2026, 11, 30), date(2026, 12, 28)},
			no:     []time.Time{date(2026, 11, 23), date(2026, 12, 21)},
		},
		{
			name:   "毎月 25 日",
			repeat: Repeat{Freq: Monthly, Interval: 1, ByMonthDay: 25},
			yes:    []time.Time{date(2026, 11, 25), date(2027, 2, 25)},
			no:     []time.Time{date(2026, 11, 24)},
		},
		{
			name:   "毎月 31 日",
			repeat: Repeat{Freq: Monthly, Interval: 1, ByMonthDay: 31},
			yes:    []time.Time{date(2026, 12, 31)},
			// 11 月に 31 日はないので、11/30 にずらさず出さない。
			no: []time.Time{date(2026, 11, 30), date(2026, 12, 1)},
		},
		{
			name:   "start と until の間だけ",
			repeat: Repeat{Freq: Daily, Interval: 1, Start: date(2026, 11, 10), Until: date(2026, 11, 20)},
			yes:    []time.Time{date(2026, 11, 10), date(2026, 11, 20)},
			no:     []time.Time{date(2026, 11, 9), date(2026, 11, 21)},
		},
	}
	for _, tt := range tests {
		for _, d := range tt.yes {
			if !tt.repeat.Occurs(d) {
				t.Errorf("%s: Occurs(%s) = false, want true", tt.name, d.Format("2006-01-02 Mon"))
			}
		}
		for _, d := range tt.no {
			if tt.repeat.Occurs(d) {
				t.Errorf("%s: Occurs(%s) = true, want false", tt.name, d.Format("2006-01-02 Mon"))
			}
		}
	}
}

// 時刻やタイムゾーンが付いていても、日付だけで決まる。
func TestOccursIgnoresTime(t *testing.T) {
	r := Repeat{Freq: Daily, Interval: 1, Until: date(2026, 11, 20)}
	late := time.Date(2026, 11, 20, 23, 30, 0, 0, time.FixedZone("JST", 9*60*60))
	if !r.Occurs(late) {
		t.Error("until の日の夜に Occurs = false")
	}
}

func weekdays(ws ...time.Weekday) []DaySpec {
	specs := make([]DaySpec, len(ws))
	for i, w := range ws {
		specs[i] = DaySpec{Weekday: w}
	}
	return specs
}

package items

import (
	"time"

	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// Occurs は、その日に定期的なタスクの予定を出すかを返す。
// 日付だけを見て、時刻とタイムゾーンは無視する。
func (r Repeat) Occurs(date time.Time) bool {
	d := yamlfile.DateOnly(date)
	if !r.Start.IsZero() && d.Before(yamlfile.DateOnly(r.Start)) {
		return false
	}
	if !r.Until.IsZero() && d.After(yamlfile.DateOnly(r.Until)) {
		return false
	}

	switch r.Freq {
	case Daily:
		return true
	case Weekly:
		if !r.onWeekday(d) {
			return false
		}
		if r.Interval == 2 {
			// 週は月曜から始まる (RFC 5545 の WKST の既定)。start を含む週から 1 週おきに出す。
			weeks := int(weekStart(d).Sub(weekStart(yamlfile.DateOnly(r.Start))).Hours() / 24 / 7)
			return weeks%2 == 0
		}
		return true
	case Monthly:
		if r.ByMonthDay != 0 {
			// その月にない日 (2 月 30 日など) は出さない。RFC 5545 と同じ扱い。
			return d.Day() == r.ByMonthDay
		}
		for _, spec := range r.ByDay {
			if spec.Weekday != d.Weekday() {
				continue
			}
			if spec.N == -1 {
				// 7 日後が翌月なら、その月の最後のその曜日。
				return d.AddDate(0, 0, 7).Month() != d.Month()
			}
			return (d.Day()-1)/7+1 == spec.N
		}
		return false
	}
	return false
}

func (r Repeat) onWeekday(d time.Time) bool {
	for _, spec := range r.ByDay {
		if spec.Weekday == d.Weekday() {
			return true
		}
	}
	return false
}

// weekStart は、その日を含む週の月曜を返す。
func weekStart(d time.Time) time.Time {
	offset := (int(d.Weekday()) + 6) % 7
	return d.AddDate(0, 0, -offset)
}

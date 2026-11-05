// Package effort はその日の工数を計算する。
package effort

import "github.com/crepidoma/lumetio/internal/dayfile"

// Task は 1 つのタスクの記録と、その時間。
type Task struct {
	dayfile.Task
	Minutes int
	// Untimed は時間不定 (時間の記録がない)。
	Untimed bool
	// Counted は工数に計上する時間がある (計上しないタスクでも、時間不定でもない)。
	Counted bool
}

// Project はプロジェクトごとのまとまり。
type Project struct {
	// Name は、そのコードで最初に出てきた記録の名前。名前がなければコード。
	Name string
	Code string
	// Minutes は計上するタスクの時間の合計。
	Minutes int
	// Counted は計上する時間がある。
	Counted bool
	// HasUntimed は、計上するタスクに時間不定のものが混ざっている。
	HasUntimed bool
	Tasks      []Task
}

// Summary はその日の工数。
type Summary struct {
	Tasks        []Task
	Projects     []Project
	TotalMinutes int
}

// Summarize はその日の記録から工数を計算する。
func Summarize(day dayfile.Day) Summary {
	var s Summary
	index := map[string]int{}
	for _, t := range day.Tasks {
		et := Task{Task: t, Untimed: len(t.Time) == 0}
		for _, e := range t.Time {
			et.Minutes += e.Minutes
		}
		et.Counted = !t.Exclude && !et.Untimed
		s.Tasks = append(s.Tasks, et)

		// 工数システムへの取り込みはコードで行うので、コードでまとめる。
		// 名前が途中で変わっても同じ付け先になり、名前が同じでもコードが違えば別になる。
		code := t.Project.Code
		i, ok := index[code]
		if !ok {
			i = len(s.Projects)
			index[code] = i
			s.Projects = append(s.Projects, Project{Name: t.Project.DisplayName(), Code: code})
		}
		p := &s.Projects[i]
		p.Tasks = append(p.Tasks, et)
		if et.Counted {
			p.Minutes += et.Minutes
			p.Counted = true
			s.TotalMinutes += et.Minutes
		}
		if et.Untimed && !t.Exclude {
			p.HasUntimed = true
		}
	}
	return s
}

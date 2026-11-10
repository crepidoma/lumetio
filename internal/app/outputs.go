package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/output"
	"github.com/crepidoma/lumetio/internal/settings"
)

// Rendered は作った出力 1 つ。
type Rendered struct {
	// Name は出力の名前。
	Name string
	output.Result
}

// RenderResult は、出力または出力セットを作った結果。
type RenderResult struct {
	Outputs []Rendered
	// Warnings は、期間の外のプロジェクトなどの警告。警告があっても出力は止めない。
	Warnings []string
	// Skipped は、読めずに除いたテンプレートの知らせ。
	Skipped []string
}

// Render は、その日の出力を作る。
// name は出力か出力セットの名前で、空なら既定の出力。
// 予定はファイルにないので、出力に入らない。
func (a *App) Render(date time.Time, name string) (RenderResult, error) {
	j, err := a.loadJob(date, name)
	if err != nil {
		return RenderResult{}, err
	}

	// すべて作ってから返す。1 つでも作れなければ、書き出しもしない。
	result := RenderResult{
		Warnings: append(outOfPeriodWarnings(j.day, j.settings.Projects), missingTimeWarnings(j.day)...),
		Skipped:  j.skipped,
	}
	for _, o := range j.outputs {
		r, err := o.Render(j.day, j.settings.Notes)
		if err != nil {
			return RenderResult{}, inputErrorf("出力 %s を作れません: %v", o.Name, err)
		}
		result.Outputs = append(result.Outputs, Rendered{Name: o.Name, Result: r})
	}
	return result, nil
}

// job は、出力を作るのに要るもの。
type job struct {
	settings settings.Settings
	outputs  []output.Output
	day      dayfile.Day
	// skipped は、読めずに除いたテンプレートの知らせ。
	skipped []string
}

// loadJob は、設定、テンプレート、その日を読み、name から作る出力を選ぶ。
func (a *App) loadJob(date time.Time, name string) (job, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return job{}, err
	}
	lib, skipped, err := a.store.LoadTemplates()
	if err != nil {
		return job{}, err
	}
	outputs, err := pick(lib, name, st.DefaultOutput)
	if err != nil {
		return job{}, err
	}
	day, _, err := a.store.LoadDay(date)
	if err != nil {
		return job{}, err
	}
	return job{settings: st, outputs: outputs, day: day, skipped: errorStrings(skipped)}, nil
}

// OutputDir は出力先のフォルダを返す。
func (a *App) OutputDir() string {
	return a.outputDir
}

// OutputDirToOpen は、ファイラーで開く出力先のフォルダを返す。まだなければ入力の誤りにする。
// 一度も出力していないと、出力先のフォルダはまだ作られていないため。
func (a *App) OutputDirToOpen() (string, error) {
	if err := mustExist(a.outputDir, "出力先のフォルダ "+a.outputDir+" はまだありません"); err != nil {
		return "", err
	}
	return a.outputDir, nil
}

// OutputFile は、name (出力か出力セットの名前。空なら既定の出力) の中の出力 outputName を、出力先に書いたときのファイルの場所を返す。
// ファイルの名前は出力するときと同じ決まりで作り、画面から受け取らない。画面から渡された場所を開くと、出力先の外も開けてしまうため。
// まだ出力していなければ、入力の誤りにする。
func (a *App) OutputFile(date time.Time, name, outputName string) (string, error) {
	j, err := a.loadJob(date, name)
	if err != nil {
		return "", err
	}
	outputName = cleanName(outputName)
	for _, o := range j.outputs {
		if o.Name != outputName {
			continue
		}
		r, err := o.Render(j.day, j.settings.Notes)
		if err != nil {
			return "", inputErrorf("出力 %s を作れません: %v", o.Name, err)
		}
		path := filepath.Join(a.outputDir, r.FileName)
		if err := mustExist(path, r.FileName+" はまだ出力していません"); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", inputErrorf("出力 %s がありません", outputName)
}

// mustExist は、path がまだなければ、message を入力の誤りとして返す。
func mustExist(path, message string) error {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &InputError{Message: message}
	}
	return err
}

// Choices は、画面で選べる出力セットと出力。
type Choices struct {
	Sets    []string
	Outputs []string
	// Default は既定の出力か出力セット。設定がないか、読めないものなら空。
	Default string
	// Skipped は、読めずに除いたテンプレートの知らせ。
	Skipped []string
}

// Choices は、画面で選べる出力セットと出力の名前を、ファイル名の順に返す。
func (a *App) Choices() (Choices, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return Choices{}, err
	}
	lib, skipped, err := a.store.LoadTemplates()
	if err != nil {
		return Choices{}, err
	}
	c := Choices{Skipped: errorStrings(skipped)}
	def := st.DefaultOutput
	for _, set := range lib.Sets {
		c.Sets = append(c.Sets, set.Name)
	}
	for _, o := range lib.Outputs {
		c.Outputs = append(c.Outputs, o.Name)
	}
	if slices.Contains(c.Sets, def) || slices.Contains(c.Outputs, def) {
		c.Default = def
	}
	return c, nil
}

// PreviewOutput は、プレビューの出力 1 つ。
type PreviewOutput struct {
	Name     string
	FileName string
	Text     string
	// Err は作れなかった理由。作れたら空。
	Err string
}

// PreviewResult は、出力または出力セットのプレビュー。
type PreviewResult struct {
	Outputs []PreviewOutput
	// Skipped は、読めずに除いたテンプレートの知らせ。
	Skipped []string
}

// Preview は、出力または出力セットを、出力ごとに作る。
// Render と違い、作れない出力があっても、その理由を入れてほかの出力を作る。
// 1 つのテンプレートの誤りで、ほかの出力まで見られなくならないようにするため。
func (a *App) Preview(date time.Time, name string) (PreviewResult, error) {
	j, err := a.loadJob(date, name)
	if err != nil {
		return PreviewResult{}, err
	}
	result := PreviewResult{Skipped: j.skipped}
	for _, o := range j.outputs {
		result.Outputs = append(result.Outputs, previewOf(o, j.day, j.settings.Notes))
	}
	return result, nil
}

// previewOf は、出力 1 つをその日の記録で作る。作れなければ、その理由を入れる。
func previewOf(o output.Output, day dayfile.Day, notes []settings.NoteDef) PreviewOutput {
	r, err := o.Render(day, notes)
	if err != nil {
		return PreviewOutput{Name: o.Name, Err: err.Error()}
	}
	return PreviewOutput{Name: o.Name, FileName: r.FileName, Text: r.Text}
}

// Export は、出力セットを作って出力先に書き、書いたファイルの名前を返す。
// name が空なら既定の出力。
// 出力先に同じ名前のファイルがあるときの扱いは conflict で決める。
func (a *App) Export(date time.Time, name string, conflict output.Conflict) (written []string, result RenderResult, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	result, err = a.Render(date, name)
	if err != nil {
		return nil, RenderResult{}, err
	}
	results := make([]output.Result, len(result.Outputs))
	for i, r := range result.Outputs {
		// 同じファイル名の出力が 2 つあると、後のもので先のものを消してしまう。
		// 大文字と小文字だけの違いも、Windows や macOS では同じファイルになる。出力の設定の誤りなので、入力の誤りとして返す。
		for _, prev := range result.Outputs[:i] {
			if strings.EqualFold(prev.FileName, r.FileName) {
				return nil, RenderResult{}, inputErrorf("出力 %s と %s が、同じファイル名 %s になります", prev.Name, r.Name, r.FileName)
			}
		}
		results[i] = r.Result
	}
	written, err = output.Write(a.outputDir, results, conflict, a.now())
	return written, result, err
}

// pick は、名前から作る出力を選ぶ。
func pick(lib output.Library, name, defaultName string) ([]output.Output, error) {
	if name == "" {
		if defaultName == "" {
			return nil, inputErrorf("既定の出力がありません。出力の名前を指定します。使える名前: %s", libraryNames(lib))
		}
		name = defaultName
	}
	// 出力や出力セットの名前は NFC で読むので、指定もそろえる。
	name = cleanName(name)
	if o, ok := lib.Output(name); ok {
		return []output.Output{o}, nil
	}
	for _, s := range lib.Sets {
		if s.Name == name {
			outputs, err := lib.SetOutputs(name)
			if err != nil {
				return nil, &InputError{Message: err.Error()}
			}
			return outputs, nil
		}
	}
	return nil, inputErrorf("出力または出力セット %s がないか、読めません。使える名前: %s", name, libraryNames(lib))
}

func libraryNames(lib output.Library) string {
	var names []string
	for _, s := range lib.Sets {
		names = append(names, s.Name)
	}
	for _, o := range lib.Outputs {
		names = append(names, o.Name)
	}
	if len(names) == 0 {
		return "なし"
	}
	return strings.Join(names, "、")
}

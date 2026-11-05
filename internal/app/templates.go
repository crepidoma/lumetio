package app

import (
	"slices"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/output"
	"github.com/crepidoma/lumetio/internal/settings"
)

// TemplateEntry は、templates/ にある出力か出力セット 1 つ。
type TemplateEntry struct {
	FileName string
	Name     string
	Kind     output.Kind
	// Data は読んだときの中身。保存や削除の前に、他で変わっていないかを比べるのに使う。
	Data string
	// Err は読めない理由。読めれば空。読めないものは、画面では消すことだけできる。
	Err string
	// Text、CSV、Set は、読めたときの中身。種類に合うものだけが入る。
	Text *output.TextSpec
	CSV  *output.CSVSpec
	Set  []string
}

// Templates は、設定画面で直す出力と出力セット。
type Templates struct {
	// Entries はファイル名の順に並ぶ。
	Entries []TemplateEntry
	// Default は settings.yaml の既定の出力。
	Default string
	// Skipped は、ファイルを読めずに一覧に入れられなかった知らせ。
	Skipped []string
}

// Templates は、templates/ の出力と出力セットを、読めないものも含めて返す。
func (a *App) Templates() (Templates, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return Templates{}, err
	}
	files, skipped, err := a.store.TemplateFiles()
	if err != nil {
		return Templates{}, err
	}
	return Templates{Entries: templateEntries(files), Default: st.DefaultOutput, Skipped: errorStrings(skipped)}, nil
}

// templateEntries は、名前の形に合うファイルを一覧にする。読めるかは、出力を作るときと同じ決まりで決める。
func templateEntries(files []output.File) []TemplateEntry {
	count := map[string]int{}
	for _, f := range files {
		if name, _, ok := output.KindOf(f.Name); ok {
			count[name]++
		}
	}
	var entries []TemplateEntry
	for _, f := range files {
		name, kind, ok := output.KindOf(f.Name)
		if !ok {
			continue
		}
		e := TemplateEntry{FileName: f.Name, Name: name, Kind: kind, Data: string(f.Data)}
		if count[name] > 1 {
			e.Err = "名前 " + name + " のファイルがほかにもあります"
			entries = append(entries, e)
			continue
		}
		if err := decodeEntry(&e, f.Data); err != nil {
			e.Err = err.Error()
			e.Text, e.CSV, e.Set = nil, nil, nil
		}
		entries = append(entries, e)
	}
	return entries
}

// decodeEntry は、ファイルを出力として読めるか確かめ、画面で直す形にする。
func decodeEntry(e *TemplateEntry, data []byte) error {
	if e.Kind == output.KindSet {
		if _, err := output.ParseSet(e.Name, data); err != nil {
			return err
		}
		var err error
		e.Set, err = output.DecodeSet(data)
		return err
	}
	if _, err := output.Parse(e.Kind, e.Name, data); err != nil {
		return err
	}
	if e.Kind == output.KindCSV {
		s, err := output.DecodeCSV(data)
		e.CSV = &s
		return err
	}
	s, err := output.DecodeText(data)
	e.Text = &s
	return err
}

// TemplateDraft は、画面で直した出力か出力セットの中身。種類に合うものだけを入れる。
type TemplateDraft struct {
	Name string
	Kind output.Kind
	Text *output.TextSpec
	CSV  *output.CSVSpec
	Set  []string
}

// marshal は、下書きをファイルの中身にし、出力として読めるかを確かめる。誤りは入力の誤りにする。
func (d TemplateDraft) marshal() (name string, data []byte, err error) {
	name = cleanName(d.Name)
	if err := output.CheckName(name); err != nil {
		return "", nil, inputErrorf("%v", err)
	}
	switch {
	case d.Kind == output.KindText && d.Text != nil:
		data, err = output.MarshalText(*d.Text)
		if err == nil {
			_, err = output.Parse(d.Kind, name, data)
		}
	case d.Kind == output.KindCSV && d.CSV != nil:
		data, err = output.MarshalCSV(*d.CSV)
		if err == nil {
			_, err = output.Parse(d.Kind, name, data)
		}
	case d.Kind == output.KindSet:
		set := make([]string, len(d.Set))
		for i, s := range d.Set {
			set[i] = cleanName(s)
		}
		data, err = output.MarshalSet(set)
		if err == nil {
			_, err = output.ParseSet(name, data)
		}
	default:
		return "", nil, inputErrorf("種類 %q の中身がありません", d.Kind)
	}
	if err != nil {
		return "", nil, inputErrorf("%v", err)
	}
	return name, data, nil
}

// SaveTemplate は、出力か出力セットを書き、書いたファイル名を返す。
// baseFile と baseData は、直し始めたときのファイル名と中身。新しく作るときは baseFile を空にする。
// ファイルが開いたときから変わっていたら、書かずに ErrConflict を返す。
// 名前を変えたら、出力セットの中の名前と既定の出力も追いかける。指す先がなくならないようにするため。
func (a *App) SaveTemplate(baseFile, baseData string, d TemplateDraft) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	name, data, err := d.marshal()
	if err != nil {
		return "", err
	}
	files, _, err := a.store.TemplateFiles()
	if err != nil {
		return "", err
	}
	entries := templateEntries(files)

	var oldName string
	if baseFile != "" {
		base, ok := findEntry(entries, baseFile)
		if !ok || base.Data != baseData {
			return "", ErrConflict
		}
		if base.Kind != d.Kind {
			return "", inputErrorf("種類は変えられません")
		}
		oldName = base.Name
	}
	// 大文字と小文字だけが違う名前も重なりとする。Windows や macOS では同じファイルになり、ほかの出力を上書きしてしまうため。
	for _, e := range entries {
		if e.FileName != baseFile && strings.EqualFold(e.Name, name) {
			return "", inputErrorf("名前 %s は、ほかの出力か出力セットで使っています", name)
		}
	}
	if d.Kind == output.KindSet {
		for _, o := range d.Set {
			if !slices.ContainsFunc(entries, func(e TemplateEntry) bool {
				return e.Name == cleanName(o) && e.Kind != output.KindSet && e.Err == ""
			}) {
				return "", inputErrorf("出力 %s がないか、読めません", o)
			}
		}
	}

	newFile := output.FileName(name, d.Kind)
	if err := a.store.WriteTemplate(newFile, data); err != nil {
		return "", err
	}
	if baseFile == "" || name == oldName {
		return newFile, nil
	}
	// 大文字と小文字だけの違いは、Windows などでは同じファイルなので、消すと書いたばかりのファイルが消える。
	sameFile := strings.EqualFold(newFile, baseFile)
	// rollback は、書いた新しいファイルを元に戻す。名前を変えられなかったときに、古い名前と新しい名前の出力が両方残らないようにするため。
	// 大文字と小文字だけの違いなら、同じファイルを書き換えたので、元の名前で元の中身を書き直す。
	// 出力の名前はファイル名から読むので、元の名前で書かないと、名前だけが変わったまま残るため。
	rollback := func() {
		if sameFile {
			_ = a.store.WriteTemplate(baseFile, []byte(baseData))
		} else {
			_ = a.store.RemoveTemplate(newFile)
		}
	}
	undo, err := a.followRename(entries, d.Kind, oldName, name)
	if err != nil {
		rollback()
		return "", err
	}
	if !sameFile {
		if err := a.store.RemoveTemplate(baseFile); err != nil {
			undo()
			rollback()
			return "", err
		}
	}
	return newFile, nil
}

// followRename は、名前を変えた出力を指す出力セットと既定の出力を、新しい名前にする。
// 返す undo は、書き換えたものを書く前に戻す。この後で名前を変えられなかったときに使う。
// 途中で失敗したら、書き換えたものを戻してから返す。名前を変えた出力は元の名前で残るので、指す側も元の名前でないと使えなくなるため。
func (a *App) followRename(entries []TemplateEntry, kind output.Kind, oldName, newName string) (undo func(), err error) {
	var steps []func()
	revert := func() {
		for _, u := range slices.Backward(steps) {
			u()
		}
	}
	defer func() {
		if err != nil {
			revert()
		}
	}()
	changed, err := a.replaceDefault(oldName, newName)
	if err != nil {
		return nil, err
	}
	if changed {
		steps = append(steps, func() { _, _ = a.replaceDefault(newName, oldName) })
	}
	if kind == output.KindSet {
		return revert, nil
	}
	for _, e := range entries {
		i := slices.Index(e.Set, oldName)
		if e.Kind != output.KindSet || e.Err != "" || i < 0 {
			continue
		}
		set := slices.Clone(e.Set)
		set[i] = newName
		data, err := output.MarshalSet(set)
		if err != nil {
			return nil, err
		}
		if err := a.store.WriteTemplate(e.FileName, data); err != nil {
			return nil, err
		}
		steps = append(steps, func() { _ = a.store.WriteTemplate(e.FileName, []byte(e.Data)) })
	}
	return revert, nil
}

// DeleteTemplate は、出力か出力セットのファイルを消す。
// baseData は開いたときの中身。変わっていたら、消さずに ErrConflict を返す。
// 出力セットに入っている出力は消さない。消すと、その出力セットが使えなくなるため。
func (a *App) DeleteTemplate(fileName, baseData string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	files, _, err := a.store.TemplateFiles()
	if err != nil {
		return err
	}
	entries := templateEntries(files)
	e, ok := findEntry(entries, fileName)
	if !ok || e.Data != baseData {
		return ErrConflict
	}
	if e.Kind != output.KindSet {
		var sets []string
		for _, s := range entries {
			if s.Kind == output.KindSet && slices.Contains(s.Set, e.Name) {
				sets = append(sets, s.Name)
			}
		}
		if len(sets) > 0 {
			return inputErrorf("出力セット %s に入っているので消せません", strings.Join(sets, "、"))
		}
	}
	if err := a.store.RemoveTemplate(fileName); err != nil {
		return err
	}
	// 消したものを既定のままにすると、指す先がなくなる。
	_, err = a.replaceDefault(e.Name, "")
	return err
}

// replaceDefault は、既定の出力が oldName なら newName にする。
func (a *App) replaceDefault(oldName, newName string) (changed bool, err error) {
	st, ver, err := a.store.LoadSettings()
	if err != nil {
		return false, err
	}
	if st.DefaultOutput != oldName {
		return false, nil
	}
	st.DefaultOutput = newName
	if _, err := a.store.SaveSettings(st, ver); err != nil {
		return false, err
	}
	return true, nil
}

func findEntry(entries []TemplateEntry, fileName string) (TemplateEntry, bool) {
	for _, e := range entries {
		if e.FileName == fileName {
			return e, true
		}
	}
	return TemplateEntry{}, false
}

// SaveDefaultOutput は既定の出力を書く。空なら既定をなくす。
// base は設定画面で読んだときの既定。他で変わっていたら書かずに ErrConflict を返す。
func (a *App) SaveDefaultOutput(base, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updateSettings(
		func(st settings.Settings) bool { return st.DefaultOutput == base },
		func(st *settings.Settings) { st.DefaultOutput = cleanName(name) },
	)
}

// PreviewTemplate は、保存する前の出力の中身で、その日の結果を作る。
// 中身に誤りがあれば、それも結果として返す。直している途中の誤りを、その場で見せるため。
func (a *App) PreviewTemplate(date time.Time, d TemplateDraft) (PreviewOutput, error) {
	if d.Kind == output.KindSet {
		return PreviewOutput{}, inputErrorf("出力セットは、出力ごとに確かめます")
	}
	name, data, err := d.marshal()
	if err != nil {
		if IsInputError(err) {
			return PreviewOutput{Name: d.Name, Err: err.Error()}, nil
		}
		return PreviewOutput{}, err
	}
	o, err := output.Parse(d.Kind, name, data)
	if err != nil {
		return PreviewOutput{Name: name, Err: err.Error()}, nil
	}
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return PreviewOutput{}, err
	}
	day, _, err := a.store.LoadDay(date)
	if err != nil {
		return PreviewOutput{}, err
	}
	return previewOf(o, day, st.Notes), nil
}

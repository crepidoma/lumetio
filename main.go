package main

import (
	"embed"
	"log"
	"os"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/gui"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/crepidoma/lumetio/internal/watch"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

// 画面に送るイベントの型を登録する。bindings に型が出て、画面で名前や中身を書き写さずに済むため。
func init() {
	application.RegisterEvent[watch.Change](gui.ChangedEvent)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err := runMCP(); err != nil {
			log.Fatal(err)
		}
		return
	}

	local, file, err := loadLocal()
	service := newService(local, file, err)

	wails := application.New(application.Options{
		Name: "lumetio",
		Services: []application.Service{
			application.NewServiceWithOptions(service, application.ServiceOptions{MarshalError: gui.MarshalError}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	win := wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "lumetio",
		Width:     defaultSize.Width,
		Height:    defaultSize.Height,
		MinWidth:  minSize.Width,
		MinHeight: minSize.Height,
		URL:       "/",
		// 前回の大きさと位置に置いてから見せる。初期の位置に一瞬出てから動くのを避けるため。
		Hidden: true,
	})
	restoreWindow(wails, win, local.Window, file)

	// 監視できなくても、他の変更が自動で画面に出ないだけで、保存するときには確かめるので、起動は続ける。
	if err := service.Watch(); err != nil {
		log.Print(err)
	}
	defer service.StopWatch()

	if err := wails.Run(); err != nil {
		log.Fatal(err)
	}
}

// loadLocal はローカル設定を読む。読めなければ、初期の値と誤りを返す。
// 読めないときは、書き換えて中身を失わないよう、file を nil にする。
func loadLocal() (localconf.Local, *localconf.File, error) {
	path, err := localconf.Path()
	if err != nil {
		return localconf.Local{}, nil, err
	}
	local, err := localconf.Load(path)
	if err != nil {
		return localconf.Local{}, nil, err
	}
	return local, localconf.NewFile(path), nil
}

// newService は、ローカル設定のデータフォルダと出力先で、画面から呼ぶ操作を作る。
// ローカル設定を読めなくても画面は開き、操作のたびにその誤りを知らせる。
func newService(local localconf.Local, file *localconf.File, err error) *gui.Service {
	env := gui.Env{
		Open:         openFile,
		Local:        file,
		Panes:        local.Panes,
		Locations:    gui.Locations{DataDir: local.DataDir, OutputDir: local.OutputDir},
		ChooseFolder: chooseFolder,
		ChooseCSV:    chooseCSV,
		Notify:       notifyChange,
	}
	// 初期値が分からなければ、設定画面で初期値に戻せないだけにする。
	if def, defErr := localconf.Defaults(); defErr == nil {
		env.DefaultLocations = gui.Locations{DataDir: def.DataDir, OutputDir: def.OutputDir}
	}
	if err != nil {
		return gui.New(nil, err, env)
	}
	return gui.New(app.New(store.New(local.DataDir), local.OutputDir, nil), nil, env)
}

// chooseFolder は、OS のフォルダを選ぶダイアログを出す。やめたら空を返す。
func chooseFolder(title, current string) (string, error) {
	return ignoreCancel(application.Get().Dialog.OpenFile().
		SetTitle(title).
		SetDirectory(current).
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		PromptForSingleSelection())
}

// chooseCSV は、OS のファイルを選ぶダイアログで CSV を選ばせる。やめたら空を返す。
func chooseCSV() (string, error) {
	return ignoreCancel(application.Get().Dialog.OpenFile().
		SetTitle("プロジェクトの CSV").
		CanChooseFiles(true).
		AddFilter("CSV", "*.csv").
		PromptForSingleSelection())
}

// dialogCancelled は、Windows でダイアログをやめたときに Wails が返す誤りの文言。
// 誤りの値は Wails の内部のパッケージにあって比べられないため、文言で見分ける。
const dialogCancelled = "cancelled by user"

// ignoreCancel は、ダイアログをやめたことを、何も選ばなかったとして返す。やめるのは失敗ではないので、知らせを出さないため。
func ignoreCancel(path string, err error) (string, error) {
	if err != nil && err.Error() == dialogCancelled {
		return "", nil
	}
	return path, err
}

// notifyChange は、データフォルダのファイルが変わったことを画面に知らせる。
func notifyChange(c watch.Change) {
	application.Get().Event.Emit(gui.ChangedEvent, c)
}

// openFile は、ファイルを OS の標準のアプリで開く。
func openFile(path string) error {
	return application.Get().Browser.OpenFile(path)
}

package localconf

import "golang.org/x/sys/windows"

// documentsDir はドキュメントのフォルダを OS から取る。
// OneDrive に移されていることがあるので、決め打ちしない。
func documentsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
}

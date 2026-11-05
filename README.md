# lumetio

日々の作業を記録して、日報のテキストと工数入力用の CSV を作るデスクトップアプリです。
記録は 1 日 1 ファイルの YAML として保存します。
保存先を OneDrive などの共有フォルダにすれば、複数の PC で同じ記録を使えます。

Windows と macOS に対応しています（macOS は動作未確認）。
画面は日本語のみです。

## 主な機能

- **記録:** その日に行ったタスクと時間を記録します。時間は `9:00-10:30` や `1h30m` のように入力できます。
- **タスク:** 直近のタスクやよく使うタスクから選んで記録できます。タスクには、プロジェクト（工数の計上先）、タグ、いつもの時間を設定できます。
- **定期的なタスク:** 繰り返しを設定したタスクは、該当する日に予定として表示されます。「実施済」にすると記録になります。
- **メモ欄:** 日ごとに書き込む欄を、自由に追加できます。欄の種類は、テキストとオン・オフの 2 つです。
- **出力:** テンプレート（Go の `text/template`）から日報のテキストを、列の定義から工数の CSV を作ります。複数の出力を出力セットにまとめて、一度に書き出すこともできます。
- **複数の PC での利用:** ほかの PC でファイルが変更されると、自動で読み込み直して画面に反映します。ほかの PC での変更を上書きすることはありません。
- **AI からの操作:** MCP サーバとしても動くので、Claude などの AI から記録の追加や出力ができます。

仕様の詳細は [設計書](documents/design.md) を、テンプレートの書き方は [テンプレートリファレンス](documents/template-reference.md) を参照してください。

## インストール

[Releases](https://github.com/crepidoma/lumetio/releases) から、お使いの OS 用のファイルをダウンロードしてください。
どちらも Microsoft や Apple のコード署名をしていないため、初回の起動時に警告が表示されます。

- **Windows:** `lumetio-<バージョン>-windows-x64.exe` を好きなフォルダに置いて起動します。「Windows によって PC が保護されました」と表示されたら、「詳細情報」を押してから「実行」を押してください。
- **macOS:** Apple シリコン搭載の Mac 用のみ提供しています。`lumetio-<バージョン>-macos-arm64.dmg` を開き、`lumetio.app` を「アプリケーション」フォルダにドラッグします。初回の起動時に「開発元を確認できない」と表示されたら、「システム設定」の「プライバシーとセキュリティ」で「このまま開く」を押すか、次のコマンドを実行してください。

  ```sh
  xattr -dr com.apple.quarantine /Applications/lumetio.app
  ```

ダウンロードしたファイルが壊れていないかは、同じ場所にある `SHA256SUMS.txt` のハッシュ値と比べて確認できます。

## データの保存場所

| 内容 | Windows | macOS |
| --- | --- | --- |
| PC ごとの設定（`local.yaml`） | `%APPDATA%\lumetio\` | `~/Library/Application Support/lumetio/` |
| データフォルダ（初期設定） | `%APPDATA%\lumetio\data\` | `~/Library/Application Support/lumetio/data/` |
| 出力先（初期設定） | ドキュメントフォルダの `lumetio\` | `~/Documents/lumetio/` |

データフォルダと出力先は、設定画面の「場所」で変更できます。
複数の PC で使う場合は、データフォルダを共有フォルダの中に作り、それぞれの PC からそのフォルダを指定してください。
ウィンドウの大きさなど PC ごとの設定は、共有されません。

データフォルダには、次のファイルが作られます。

| ファイル | 内容 |
| --- | --- |
| `days/YYYY/YYYY-MM-DD.yaml` | 日次ファイル。その日の記録とメモ欄 |
| `items.yaml` | タスクの設定（よく使う、いつもの時間、繰り返しなど） |
| `settings.yaml` | プロジェクト、メモ欄の定義、時間の表示形式、既定の出力 |
| `templates/` | 出力と出力セットの定義 |

## MCP サーバとして使う

`lumetio mcp` と実行すると、画面を開かずに MCP サーバ（stdio）として起動します。
読み書きするデータフォルダは、画面から使うときと同じです。

Claude Desktop などには、次のように登録します（パスは実際に置いた場所に合わせてください）。

```json
{
  "mcpServers": {
    "lumetio": {
      "command": "C:\\Tools\\lumetio\\lumetio.exe",
      "args": ["mcp"]
    }
  }
}
```

Claude Code の場合は、次のコマンドで登録します。

```sh
claude mcp add lumetio -- C:\Tools\lumetio\lumetio.exe mcp
```

使えるツールの一覧は、設計書の「MCP」の章にあります。

## 開発

### 環境の準備

- [mise](https://mise.jdx.dev/) を入れます。
  - Go、Node.js、pnpm は、`mise.toml` で指定したバージョンを使います（`mise install` でインストールされます）。
  - Go をグローバルに有効にしていないため、コマンドには `mise exec --` を付けて実行します。
- Wails v3 の CLI をインストールします。

  ```sh
  mise exec -- go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
  ```

- 画面のテストで使うブラウザを、最初に 1 回だけインストールします。

  ```sh
  mise exec -- pnpm --dir frontend install
  mise exec -- pnpm --dir frontend exec playwright install chromium webkit
  ```

### 実行とビルド

```sh
mise exec -- wails3 dev     # 開発モードで起動する。フロントエンドの変更がすぐに反映される
mise exec -- wails3 build   # bin/ に実行ファイルを作る
```

### テストと CI

変更を確認するためのコマンド（Go のテスト、Biome、画面のテスト、ビルド）は、[AGENTS.md](AGENTS.md) の「完了の定義」にまとめています。
PR を作成すると、GitHub Actions（`.github/workflows/ci.yml`）が同じ確認を Windows と macOS で実行し、ビルドしたアプリ（macOS は dmg）を Artifacts にアップロードします。
Artifacts のアプリの起動方法は、「インストール」と同じです。

### リリースの手順

1. バージョンを上げる PR を作成して、マージします。バージョンは次のすべてに書きます（`wails3 update build-assets` はほかの項目も初期値に戻してしまうため使いません）。
   - `build/config.yml` の `version`
   - `build/darwin/Info.plist` と `build/darwin/Info.dev.plist` の `CFBundleVersion` と `CFBundleShortVersionString`
   - `build/windows/info.json` の `file_version` と `ProductVersion`
2. マージした `main` のコミットにタグを付けて、push します。

   ```sh
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. GitHub Actions（`.github/workflows/release.yml`）が、タグとアプリのバージョンが一致するかを確認し、CI と同じテストとビルドを行ってから、リリースの下書きを作ります。
4. 下書きの内容を確認して、GitHub の画面から公開します。

### ディレクトリ構成

| 場所 | 内容 |
| --- | --- |
| `main.go`、`window.go` | アプリの起動とウィンドウ |
| `mcp.go` | `lumetio mcp` の起動 |
| `internal/app` | 画面と MCP の両方から使う処理。アプリのルールはここに書く |
| `internal/gui` | 画面から呼び出す処理（Wails のサービス） |
| `internal/mcpserver` | MCP のツール |
| `internal/store`、`internal/watch` | データフォルダの読み書きと、ファイルの変更の監視 |
| `internal/` のその他 | 時間の解釈、繰り返し、工数の計算、出力の生成など |
| `frontend/src` | 画面（React、Mantine） |
| `frontend/bindings` | Go のコードから自動生成される呼び出し用のコード。手で編集しない |
| `documents` | 設計書とテンプレートリファレンス |

開発のルール（完了の条件、コミット、用語など）は [AGENTS.md](AGENTS.md) にまとめています。

## ライセンス

[MIT](LICENSE)

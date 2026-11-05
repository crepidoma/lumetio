# lumetio

日々の作業記録から、日報のテキストと工数の CSV を作るデスクトップアプリ。
データは 1 日 1 ファイルの YAML で、OneDrive などの共有フォルダに置けば、複数の PC から同じ記録を使える。

Windows と macOS で動かす（macOS は未検証）。
画面は日本語だけ。

## できること

- **記録:** その日のタスクと時間を記録する。時間は `9:00-10:30` や `1h30m` のように書ける。
- **タスク:** 直近のタスク、よく使うタスクから選んで記録する。プロジェクト（工数を計上する先）、タグ、いつもの時間を持たせられる。
- **定期的なタスク:** 繰り返しを設定したタスクは、その日の予定として出る。実施済にすると記録になる。
- **メモ欄:** 日ごとに、テキストか真偽値の欄を自由に持てる。
- **出力:** Go の `text/template` で書いたテンプレートから日報のテキストを、列の定義から工数の CSV を作る。いくつかの出力を出力セットにまとめて、一度に書き出せる。
- **複数の PC:** 他の PC が書き換えたファイルは、読み込み直して画面に出す。他の PC の変更を上書きしない。
- **AI からの操作:** 同じ実行ファイルが MCP サーバとしても動く。

決まりの詳細は [設計書](documents/design.md)、テンプレートの書き方は [テンプレートリファレンス](documents/template-reference.md) にある。

## ファイルの置き場所

| もの | Windows | macOS |
| --- | --- | --- |
| PC ごとの設定（`local.yaml`） | `%APPDATA%\lumetio\` | `~/Library/Application Support/lumetio/` |
| データフォルダ（初期値） | `%APPDATA%\lumetio\data\` | `~/Library/Application Support/lumetio/data/` |
| 出力先（初期値） | ドキュメントの下の `lumetio\` | `~/Documents/lumetio/` |

データフォルダと出力先は、設定画面の「場所」で変えられる。
複数の PC で使うときは、データフォルダを共有フォルダの中に置き、どの PC からもそのフォルダを選ぶ。
PC ごとの設定（ウィンドウの大きさやペインの幅など）は共有しない。

データフォルダの中身は次のとおり。

| ファイル | 中身 |
| --- | --- |
| `days/YYYY/YYYY-MM-DD.yaml` | 日次ファイル。その日の記録とメモ欄 |
| `items.yaml` | タスクの設定（よく使う、いつもの時間、繰り返しなど） |
| `settings.yaml` | プロジェクト、メモ欄の定義、長さの表記、既定の出力 |
| `templates/` | 出力と出力セットの定義 |

## MCP

`lumetio mcp` で起動すると、画面を出さずに MCP サーバ（stdio）として動く。
画面と同じデータフォルダを読み書きする。

Claude Desktop などでは、次のように登録する（パスは置いた場所に合わせる）。

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

Claude Code では次のとおり。

```sh
claude mcp add lumetio -- C:\Tools\lumetio\lumetio.exe mcp
```

使えるツールは、設計書の「MCP」の節にある。

## 開発

### 用意するもの

- [mise](https://mise.jdx.dev/)
  - Go、Node.js、pnpm は、リポジトリの `mise.toml` の版を使う（`mise install` で入る）。
  - コマンドには `mise exec --` を付ける。Go はグローバルに有効化しないため。
- Wails v3 のコマンド:

  ```sh
  mise exec -- go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
  ```

- 画面のテストに使うブラウザ（初めに 1 回）:

  ```sh
  mise exec -- pnpm --dir frontend install
  mise exec -- pnpm --dir frontend exec playwright install chromium webkit
  ```

### 動かす

```sh
mise exec -- wails3 dev     # 開発モード。フロントエンドの変更がすぐ画面に出る
mise exec -- wails3 build   # bin/ に実行ファイルを作る
```

### 確かめる

変更を確かめるコマンド（Go のテスト、Biome、画面のテスト、ビルド）は、[AGENTS.md](AGENTS.md) の「完了の定義」にある。
GitHub で PR を出すか `main` に push すると、GitHub Actions（`.github/workflows/ci.yml`）が同じものを Windows と macOS で動かし、できた実行ファイル（macOS は `.app` を入れた dmg）を Artifacts に置く。

macOS 版は Apple シリコン用だけを作る。
dmg を開き、`lumetio.app` を「アプリケーション」へドラッグして入れる。
`.app` には Apple の正式な署名をしていない。
初めて開くときに「開発元を確認できない」と出たら、「システム設定」の「プライバシーとセキュリティ」で「このまま開く」を押すか、次のコマンドでダウンロードしたしるしを外す。

```sh
xattr -dr com.apple.quarantine /Applications/lumetio.app
```

### 構成

| 場所 | 中身 |
| --- | --- |
| `main.go`、`window.go` | アプリの起動とウィンドウ |
| `mcp.go` | `lumetio mcp` の起動 |
| `internal/app` | 画面と MCP の両方から使う操作。決まりはここに置く |
| `internal/gui` | 画面から呼ぶ操作（Wails のサービス） |
| `internal/mcpserver` | MCP のツール |
| `internal/store`、`internal/watch` | データフォルダの読み書きと、変わったことの知らせ |
| `internal/` のほか | 時間の解釈、繰り返し、工数の計算、出力の生成など |
| `frontend/src` | 画面（React、Mantine） |
| `frontend/bindings` | Go の定義から生成される呼び出し。手で編集しない |
| `documents` | 設計書とテンプレートリファレンス |

開発の決まり（完了の条件、コミット、用語など）は [AGENTS.md](AGENTS.md) にある。

## ライセンス

[MIT](LICENSE)

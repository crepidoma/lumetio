---
name: build-and-launch
description: lumetio をビルドし、アプリを起動してウィンドウが開くことを確かめてから終了させる。開発モードを試し用のデータで起動する手順も含む。「動かして」「起動確認して」「ビルドして」と言われたとき、画面に関わる変更の後に使う。
---

# ビルドと起動確認

## 1. ビルド

リポジトリのルートで実行する。

```sh
mise exec -- wails3 build
```

次の順に進み、Windows では `bin/lumetio.exe` ができる。

1. フロントエンドの依存を入れる（pnpm）
2. `frontend/bindings` を Go の定義から作り直す
3. `tsc` とフロントエンドのビルド
4. Go のビルド

`frontend/bindings` に差分が出たら、Go 側の変更と同じコミットに入れる。

## 2. 起動して確かめる（Windows）

起動し、ウィンドウのタイトルが `lumetio` になっていることを見てから止める。
**止め忘れない。**

```powershell
$p = Start-Process -FilePath .\bin\lumetio.exe -PassThru
Start-Sleep -Seconds 4
"alive=$(-not $p.HasExited) title=$((Get-Process -Id $p.Id).MainWindowTitle)"
Stop-Process -Id $p.Id
```

`alive=False` なら起動直後に落ちている。

## 3. 開発モード（試し用のデータで）

Vite が 9245 番で待ち受け、フロントエンドの変更がすぐ画面に出る。
**ユーザーの実データを使わないよう、PC ごとの設定の場所を scratchpad に向けて起動する。**

PC ごとの設定（`local.yaml`）は、Windows では `%APPDATA%\lumetio\`、macOS では `~/Library/Application Support/lumetio/` から読む。
Windows では `APPDATA` を変えれば、データフォルダの初期値もその下になる。

出力先の初期値は OS のドキュメントのフォルダで、`APPDATA` を変えても変わらない。
出力を試す前に、`local.yaml` の `output_dir` を scratchpad に向けておく。

```powershell
$root = "<scratchpad>\uitest"
New-Item -ItemType Directory -Force "$root\appdata\lumetio", "$root\out" | Out-Null
"output_dir: $root\out" | Set-Content -Encoding utf8 "$root\appdata\lumetio\local.yaml"
$env:APPDATA = "$root\appdata"
mise exec -- wails3 dev *> "$root\dev.log"
```

- 止めるまで終わらないので、バックグラウンドで起動する。
- **作業の最後に必ず止め**、アプリと `wails3` のプロセスが残っていないか確かめる。
- 画面の操作はこちらからはできない（ウィンドウにキーやクリックを送らない）。画面を見る確認は、ユーザーに頼む。
- 日次ファイルなどを書き換えて、他の PC の変更として届くかを試すときは、scratchpad のデータフォルダのファイルを書き換える。

## macOS

この環境からは確かめられない。
ユーザーに Mac で `wails3 build` と起動を頼む。

## よくある失敗

| 症状 | 原因 |
| --- | --- |
| `go` や `wails3` が見つからない | `mise exec --` を付けていない。Go はグローバルに有効化していない |
| `The paging file is too small for this operation to complete.` | PC のメモリ不足。`go vet` と `go test` は `-p 1` を付けて、並べて動かす数を減らす |
| `pattern all:frontend/dist: no matching files found` | フロントエンドを一度もビルドしていない状態で `go build` や `go vet` を実行した。先に `wails3 build` を通す |
| 開発モードで、Mantine の Tabs の初期値などを変えても画面が変わらない | ホットリロードは部品の状態を残す。部品の `key` を一時的に変えるか、開発モードを起動し直す |

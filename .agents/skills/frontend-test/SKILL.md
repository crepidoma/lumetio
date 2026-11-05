---
name: frontend-test
description: lumetio の画面のテストを Vitest の browser mode（Chromium と WebKit）で書いて動かす。Go の呼び出しの差し替え、DayContext の渡し方、Esc やドラッグの操作、落ちたときの調べ方を含む。画面の部品の振る舞いを確かめるとき、画面の不具合を直すときに使う。
---

# 画面のテスト

## 置き場所と動かし方

| ファイル | 動く場所 | 向くもの |
| --- | --- | --- |
| `src/<部品>.browser.test.tsx` | ヘッドレスの Chromium と WebKit | 部品の状態、effect、Go を呼んだ結果、Esc、ドラッグ |
| `src/<名前>.test.ts` | Node（ブラウザを起動しない） | 並びの計算などの関数 |

関数で試せるものは関数のテストにする。
ブラウザのテストより速い。

```sh
mise exec -- pnpm --dir frontend test                                   # すべて
mise exec -- pnpm --dir frontend exec vitest run src/X.browser.test.tsx # ファイルを選ぶ
mise exec -- pnpm --dir frontend exec vitest run --project "browser (chromium)" -t "テストの名前の一部"
```

直している途中は Chromium だけで回してよい。
終わる前に、必ず両方のブラウザで通す。

## 道具（`src/test/`）

| もの | 使い方 |
| --- | --- |
| `renderApp(ui)`（`render.tsx`） | アプリと同じテーマと知らせの置き場の中に描く |
| `Providers`（`render.tsx`） | `renderHook` の `wrapper` に使う |
| `inputError(message)`（`render.tsx`） | Go が返す入力の誤りと同じ形のエラー |
| `makeDay`、`textNote`、`dayValue`、`WithDay`（`day.tsx`） | 日次画面の部品に渡す `DayContext` を作る。`dayValue` の `act` と `run` は、渡した Go の操作をそのまま呼ぶ |
| `drag(from, to)`、`center(e)`（`drag.ts`） | 本物のマウスでドラッグする |

画面は 1280×800 で描き、アプリと同じ CSS（`setup.ts`）を読む。

## Go の呼び出しを差し替える

`frontend/bindings` は `vi.mock` で差し替える。
テストで使う関数だけを `vi.fn()` で並べる。

```tsx
vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { SetText: vi.fn(async () => ({})) },
}));

beforeEach(() => {
	vi.mocked(Service.SetText).mockClear();
});
```

- 値を返す: `vi.mocked(Service.X).mockResolvedValue(...)`。型は bindings の `models.ts` に合わせる。
- 入力の誤り: `mockRejectedValue(inputError("…"))`
- 衝突: `new Error("…", { cause: { kind: "conflict" } })`
- 読み書きの失敗: 素の `new Error("…")`

## 操作と確かめ方

- 要素は役割と名前で探す（`screen.getByRole("textbox", { name: "連絡" })`）。名前は画面の日本語の文言そのもの。名前の一部で探すときは正規表現を使う（`{ name: /^壊れた\.yaml/ }`）。
- 入力は `fill`、押すのは `click`。
- 欄から離れる（blur）のは、見出しなど別の要素の `click` で起こす。
- キーは `userEvent.keyboard("{Escape}")`（`vitest/browser`）。
- フォーカスを置くのは `(locator.element() as HTMLElement).focus()`。
- 画面に出るものは `await expect.element(locator).toBeVisible()` で待つ。決まった時間待つのは、どうしても待つ目印がないときだけにする。
- Go を呼んだかは `vi.waitFor(() => expect(Service.X).toHaveBeenCalled…)` で待つ。
- 部品を描き直すのは `screen.rerender(...)`。フックは `renderHook` で試す。
- ドラッグは `drag(取っ手, center(置き先))`。Playwright の `dragAndDrop` は、置き先が止まるまで待つので、行が動き続ける並べ替えでは終わらない。押したポインタを捕まえる部品（境目）も、`drag` なら動く。

## 落ちたとき

- 落ちたときの画面の写しが `frontend/.vitest/attachments/failure-screenshots/` に残る（Git には入れない）。Read で開いて見る。
- 原因が分からないときは、部品に一時的に `console.log` を入れて動かすと、出力がテストの結果に出る。**調べ終わったら必ず消す。**
- 片方のブラウザだけで落ちるときは、描くのを待たずに次の操作をしていないかを疑う。WebKit は遅れやすい。
- `optimized dependencies changed. reloading` と出たあと、`Vitest failed to find the runner` や `did not become ready` で落ちることがある。テストの途中で見つかった依存を、Vite が事前バンドルし直したため。出力の `dependencies optimized:` に並んだ依存を、`vitest.config.ts` の `optimizeDeps.include` に足す。手元ではキャッシュが残っていて起きないので、`frontend/node_modules/.vite` を消してから動かして確かめる。

## テストの名前と値

- テストの名前は、守る決まりを日本語で書く。
- 値は架空にする。日付は `2026-11-05` のような決まった日にし、今日に頼らない。

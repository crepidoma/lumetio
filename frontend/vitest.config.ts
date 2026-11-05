import react from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import { defineConfig } from "vitest/config";
import type { BrowserCommand } from "vitest/node";

type Point = { x: number; y: number };

// mouse は、本物のマウスを、テストの画面の中の座標 p へ動かしてから、押すか離す (move なら動かすだけ)。
// ドラッグのあいだに要素が動いても待たず、押したポインタを捕まえる部品 (境目) も、実際の操作と同じに動かすため。
// 動かすたびにテストの側で描くのを待てるよう、押す、動かす、離すを分けて呼ぶ。
const mouse: BrowserCommand<
	[action: "down" | "move" | "up", p: Point]
> = async (ctx, action, p) => {
	const frame = await ctx.frame();
	const box = await (await frame.frameElement()).boundingBox();
	// テストの画面は、ブラウザの窓に収まるよう縮めて置かれることがあるので、その倍率も掛ける。
	const width = await frame.evaluate(() => window.innerWidth);
	const scale = box ? box.width / width : 1;
	await ctx.page.mouse.move(
		(box?.x ?? 0) + p.x * scale,
		(box?.y ?? 0) + p.y * scale,
	);
	if (action === "down") await ctx.page.mouse.down();
	if (action === "up") await ctx.page.mouse.up();
};

// 関数だけのテストは Node で動かし、ブラウザを起動しない。速く回すため。
// 画面の部品のテストは、Windows の WebView2 に合わせた Chromium と、macOS の WKWebView に合わせた WebKit の両方で動かす。
export default defineConfig({
	plugins: [react()],
	test: {
		projects: [
			{
				extends: true,
				test: {
					name: "unit",
					include: ["src/**/*.test.ts"],
					environment: "node",
				},
			},
			{
				extends: true,
				test: {
					name: "browser",
					include: ["src/**/*.browser.test.tsx"],
					setupFiles: ["src/test/setup.ts"],
					browser: {
						enabled: true,
						provider: playwright(),
						headless: true,
						// デスクトップのアプリなので、ウィンドウを開いたときに近い大きさで描く。
						viewport: { width: 1280, height: 800 },
						instances: [{ browser: "chromium" }, { browser: "webkit" }],
						commands: { mouse },
					},
				},
			},
		],
	},
});

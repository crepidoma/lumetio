import { commands } from "vitest/browser";

type Point = { x: number; y: number };

declare module "vitest/browser" {
	interface BrowserCommands {
		mouse: (action: "down" | "move" | "up", p: Point) => Promise<void>;
	}
}

export function center(e: Element): Point {
	const r = e.getBoundingClientRect();
	return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
}

const frame = () => new Promise((r) => requestAnimationFrame(r));

// drag は、本物のマウスで from の真ん中を押し、to まで少しずつ動かして離す。
// 動かすたびに描き終えるのを待つ。ドラッグする部品は、描いた後の位置で置き場所を決めるので、
// 待たずに動かすと、遅いブラウザでは置き場所が決まる前に離してしまうため。
export async function drag(from: Element, to: Point) {
	const start = center(from);
	await commands.mouse("down", start);
	const steps = 10;
	for (let i = 1; i <= steps; i++) {
		await commands.mouse("move", {
			x: start.x + ((to.x - start.x) * i) / steps,
			y: start.y + ((to.y - start.y) * i) / steps,
		});
		await frame();
	}
	// 手で動かすときと同じく、置き場所を測り直すまで少し止まってから離す。
	for (let i = 0; i < 3; i++) {
		await commands.mouse("move", to);
		await frame();
	}
	await commands.mouse("up", to);
}

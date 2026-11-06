import { expect, test, vi } from "vitest";
import { userEvent } from "vitest/browser";
import { Splitter } from "./Splitter";
import { center, drag } from "./test/drag";
import { renderApp } from "./test/render";

async function renderSplitter() {
	const props = {
		onStart: vi.fn(),
		onMove: vi.fn(),
		onEnd: vi.fn(),
		onKey: vi.fn(),
	};
	const screen = await renderApp(
		<div style={{ display: "flex", height: 200 }}>
			<div style={{ width: 100 }} />
			<Splitter
				label="左ペインの幅"
				value={100}
				min={50}
				max={300}
				{...props}
			/>
		</div>,
	);
	return { screen, props };
}

test("ドラッグすると、押した位置から動かした量を渡す", async () => {
	const { screen, props } = await renderSplitter();
	const splitter = screen
		.getByRole("separator", { name: "左ペインの幅" })
		.element();
	const from = center(splitter);
	await drag(splitter, { x: from.x + 40, y: from.y });
	expect(props.onStart).toHaveBeenCalledOnce();
	expect(props.onMove).toHaveBeenLastCalledWith(40);
	expect(props.onEnd).toHaveBeenCalledOnce();
});

test("表示を拡大していて位置が小数になっても、動かした量は整数で渡す", async () => {
	const { screen, props } = await renderSplitter();
	const splitter = screen
		.getByRole("separator", { name: "左ペインの幅" })
		.element();
	const from = center(splitter);
	await drag(splitter, { x: from.x + 40.5, y: from.y });
	const dx = props.onMove.mock.lastCall?.[0];
	expect(Number.isInteger(dx)).toBe(true);
});

test("左右キーで、動かす向きを渡す", async () => {
	const { screen, props } = await renderSplitter();
	const splitter = screen
		.getByRole("separator", { name: "左ペインの幅" })
		.element() as HTMLElement;
	splitter.focus();
	await userEvent.keyboard("{ArrowLeft}");
	await userEvent.keyboard("{ArrowRight}");
	expect(props.onKey.mock.calls).toEqual([[-1], [1]]);
});

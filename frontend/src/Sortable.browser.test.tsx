import { useState } from "react";
import { expect, test, vi } from "vitest";
import { userEvent } from "vitest/browser";
import { SortableItem, SortableList } from "./Sortable";
import { center, drag } from "./test/drag";
import { renderApp } from "./test/render";

function List({ onReorder }: { onReorder: (ids: string[]) => void }) {
	const [ids, setIds] = useState(["一", "二", "三"]);
	return (
		<SortableList
			ids={ids}
			onReorder={async (next) => {
				onReorder(next);
				setIds(next);
			}}
		>
			{(order) =>
				order.map((id) => (
					<SortableItem key={id} id={id} label={id}>
						{(handle) => (
							<div style={{ display: "flex", height: 40 }}>
								{handle}
								<span>{id}</span>
							</div>
						)}
					</SortableItem>
				))
			}
		</SortableList>
	);
}

test("取っ手をドラッグして、置いた行の位置へ動かす", async () => {
	const onReorder = vi.fn();
	const screen = await renderApp(<List onReorder={onReorder} />);
	await drag(
		screen.getByRole("button", { name: "三を並べ替える" }).element(),
		center(screen.getByText("一").element()),
	);
	await vi.waitFor(() =>
		expect(onReorder).toHaveBeenCalledExactlyOnceWith(["三", "一", "二"]),
	);
});

test("キーボードでつかんで動かし、置く", async () => {
	const onReorder = vi.fn();
	const screen = await renderApp(<List onReorder={onReorder} />);
	const row = (id: string) =>
		screen.getByText(id).element().closest(".sortable") as HTMLElement;
	const secondTop = row("二").getBoundingClientRect().top;
	(
		screen
			.getByRole("button", { name: "一を並べ替える" })
			.element() as HTMLElement
	).focus();
	await userEvent.keyboard("{Space}");
	await vi.waitFor(() => expect(row("一")).toHaveClass("dragging"));
	// 並べ替えの部品は、つかんだ後に行の位置を測り終えるまで、上下キーを捨てる。
	// 測り終えたことは画面に出ないので、下の行が上へずれる (置き場所が下の行に移る) まで押し直す。
	// 移る前に Space で置くと、元の位置に置いたことになる。
	for (let i = 0; ; i++) {
		await userEvent.keyboard("{ArrowDown}");
		try {
			await vi.waitFor(() =>
				expect(row("二").getBoundingClientRect().top).toBeLessThan(secondTop),
			);
			break;
		} catch (e) {
			if (i >= 4) throw e;
		}
	}
	await userEvent.keyboard("{Space}");
	await vi.waitFor(() =>
		expect(onReorder).toHaveBeenCalledExactlyOnceWith(["二", "一", "三"]),
	);
});

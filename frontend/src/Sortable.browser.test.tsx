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
	(
		screen
			.getByRole("button", { name: "一を並べ替える" })
			.element() as HTMLElement
	).focus();
	await userEvent.keyboard("{Space}");
	await userEvent.keyboard("{ArrowDown}");
	await userEvent.keyboard("{Space}");
	await vi.waitFor(() =>
		expect(onReorder).toHaveBeenCalledExactlyOnceWith(["二", "一", "三"]),
	);
});

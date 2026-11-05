import { useState } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { userEvent } from "vitest/browser";
import {
	type Record,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { RecordRow } from "./RecordRow";
import { dayValue, makeDay, WithDay } from "./test/day";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { RemoveRecord: vi.fn(async () => ({})) },
}));

beforeEach(() => {
	vi.mocked(Service.RemoveRecord).mockClear();
});

const record: Record = {
	title: "調査",
	project: null,
	outOfPeriod: false,
	tags: [],
	note: "",
	exclude: false,
	done: false,
	times: [],
	total: "",
};

function Row() {
	const [open, setOpen] = useState<string | null>("調査");
	return (
		<WithDay value={dayValue(makeDay("2026-11-05"))}>
			<RecordRow record={record} open={open === "調査"} onOpen={setOpen} />
		</WithDay>
	);
}

async function askRemove() {
	const screen = await renderApp(<Row />);
	await screen.getByRole("button", { name: "記録を消す" }).click();
	await expect
		.element(screen.getByText("調査 の記録を消しますか？"))
		.toBeVisible();
	expect(Service.RemoveRecord).not.toHaveBeenCalled();
	return screen;
}

test("記録を消す前に確かめ、続けたら消す", async () => {
	const screen = await askRemove();
	await screen.getByRole("button", { name: "消す", exact: true }).click();
	expect(Service.RemoveRecord).toHaveBeenCalledExactlyOnceWith(
		"2026-11-05",
		"調査",
	);
});

test("確かめてキャンセルしたら消さない", async () => {
	const screen = await askRemove();
	await screen.getByRole("button", { name: "キャンセル" }).click();
	await expect
		.element(screen.getByText("調査 の記録を消しますか？"))
		.not.toBeInTheDocument();
	expect(Service.RemoveRecord).not.toHaveBeenCalled();
	// 詳しい表示は開いたまま。
	await expect
		.element(screen.getByRole("button", { name: "記録を消す" }))
		.toBeVisible();
});

test("確かめるダイアログの Esc は、ダイアログだけを閉じる", async () => {
	const screen = await askRemove();
	await userEvent.keyboard("{Escape}");
	await expect
		.element(screen.getByText("調査 の記録を消しますか？"))
		.not.toBeInTheDocument();
	expect(Service.RemoveRecord).not.toHaveBeenCalled();
	await expect
		.element(screen.getByRole("button", { name: "記録を消す" }))
		.toBeVisible();
});

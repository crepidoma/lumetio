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
	Service: {
		RemoveRecord: vi.fn(async () => ({})),
		UpdateRecord: vi.fn(async () => ({})),
	},
}));

beforeEach(() => {
	vi.mocked(Service.RemoveRecord).mockClear();
	vi.mocked(Service.UpdateRecord).mockClear();
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
	// 時間未入力の記録。Go は合計に 0 を長さの表記で入れて返す。
	total: "0:00",
	untimed: false,
};

function Row({ value = record }: { value?: Record }) {
	const [open, setOpen] = useState<string | null>("調査");
	return (
		<WithDay
			value={dayValue(makeDay("2026-11-05"), {
				tags: ["ふりかえり", "調査", "Review"],
			})}
		>
			<RecordRow record={value} open={open === "調査"} onOpen={setOpen} />
		</WithDay>
	);
}

const timed: Record = {
	...record,
	times: [{ label: "0:30", input: "0:30", value: "30m" }],
	total: "0:30",
};

test("時間不定の記録は、合計の位置に「不定」と出す", async () => {
	const screen = await renderApp(<Row value={{ ...record, untimed: true }} />);
	await expect.element(screen.getByText("不定", { exact: true })).toBeVisible();
});

test("時間未入力の記録は、合計の位置に 0 の時間を出す", async () => {
	const screen = await renderApp(<Row />);
	await expect.element(screen.getByText("0:00", { exact: true })).toBeVisible();
	await expect
		.element(screen.getByText("不定", { exact: true }))
		.not.toBeInTheDocument();
});

test("詳しい表示の切り替えは、時間不定、計上しない、完了の順に並べる", async () => {
	const screen = await renderApp(<Row />);
	const labels = screen
		.getByRole("checkbox")
		.elements()
		.map((e) => document.querySelector(`label[for="${e.id}"]`)?.textContent);
	expect(labels).toEqual(["時間不定", "計上しない", "完了"]);
});

test("計上しないと完了の札は、タグと見分けられる形で出す", async () => {
	const screen = await renderApp(
		<Row value={{ ...record, tags: ["調査"], exclude: true, done: true }} />,
	);
	const tag = screen.getByText("#調査").element();
	for (const label of ["計上しない", "完了"]) {
		const badge = screen.getByText(label, { exact: true }).first().element();
		expect(badge.className).not.toBe(tag.className);
	}
});

test("詳しい表示で、時間のない記録を時間不定にする", async () => {
	const screen = await renderApp(<Row />);
	await screen.getByRole("checkbox", { name: "時間不定" }).click();
	await vi.waitFor(() =>
		expect(Service.UpdateRecord).toHaveBeenCalledExactlyOnceWith(
			"2026-11-05",
			"調査",
			{ untimed: true },
		),
	);
});

test("時間がある記録は、時間不定にできない", async () => {
	const screen = await renderApp(<Row value={timed} />);
	await expect
		.element(screen.getByRole("checkbox", { name: "時間不定" }))
		.toBeDisabled();
	await expect.element(screen.getByText("0:30").first()).toBeVisible();
});

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

test("タグの欄は、タグの一覧から打った文字を含むものを候補に出す", async () => {
	const screen = await renderApp(<Row />);
	await screen.getByRole("combobox", { name: "タグ" }).fill("rev");
	// 大文字と小文字は区別しない。
	await expect
		.element(screen.getByRole("option", { name: "Review" }))
		.toBeVisible();
	await expect
		.element(screen.getByRole("option", { name: "ふりかえり" }))
		.not.toBeInTheDocument();
	await screen.getByRole("option", { name: "Review" }).click();
	await vi.waitFor(() =>
		expect(Service.UpdateRecord).toHaveBeenCalledExactlyOnceWith(
			"2026-11-05",
			"調査",
			{ tags: ["Review"] },
		),
	);
});

test("一覧にないタグも、打てば付けられる", async () => {
	const screen = await renderApp(<Row />);
	await screen.getByRole("combobox", { name: "タグ" }).fill("新しいタグ");
	await userEvent.keyboard("{Enter}");
	await vi.waitFor(() =>
		expect(Service.UpdateRecord).toHaveBeenCalledExactlyOnceWith(
			"2026-11-05",
			"調査",
			{ tags: ["新しいタグ"] },
		),
	);
});

const middle = (r: DOMRect) => (r.top + r.bottom) / 2;

test("札はタスク名の下の段に出し、合計はその 2 段の高さの真ん中に出す", async () => {
	const screen = await renderApp(
		<Row value={{ ...timed, tags: ["調査"], exclude: true }} />,
	);
	const title = screen
		.getByRole("button", { name: "調査", exact: true })
		.element()
		.getBoundingClientRect();
	const badges = (
		screen.getByText("#調査").element().closest(".row-badges") as HTMLElement
	).getBoundingClientRect();
	const total = (
		document.querySelector(".row-total") as HTMLElement
	).getBoundingClientRect();
	expect(badges.top).toBeGreaterThanOrEqual(title.bottom);
	expect(
		Math.abs(middle(total) - (title.top + badges.bottom) / 2),
	).toBeLessThan(2);
});

test("札がなければ 2 段目は出さず、合計はタスク名の高さの真ん中に出す", async () => {
	await renderApp(<Row value={timed} />);
	expect(document.querySelector(".row-badges")).toBeNull();
	const title = (
		document.querySelector(".row-title") as HTMLElement
	).getBoundingClientRect();
	const total = (
		document.querySelector(".row-total") as HTMLElement
	).getBoundingClientRect();
	expect(Math.abs(middle(total) - middle(title))).toBeLessThan(2);
});

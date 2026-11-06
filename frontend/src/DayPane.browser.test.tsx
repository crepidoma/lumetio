import { expect, test, vi } from "vitest";
import { DayPane } from "./DayPane";
import { dayValue, makeDay, WithDay } from "./test/day";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: {},
}));

test("記録の一覧と予定の一覧のあいだも、記録どうしと同じ間をあける", async () => {
	const day = {
		...makeDay("2026-11-05"),
		records: ["調査", "設計"].map((title) => ({
			title,
			project: null,
			outOfPeriod: false,
			tags: [],
			note: "",
			exclude: false,
			done: false,
			times: [],
			total: "",
			untimed: false,
		})),
		plans: [{ title: "朝会", project: null, times: [], untimed: false }],
	};
	const screen = await renderApp(
		<WithDay value={dayValue(day)}>
			<DayPane />
		</WithDay>,
	);
	const rowOf = (title: string) =>
		(screen.getByText(title, { exact: true }).element() as HTMLElement)
			.closest(".row")
			?.getBoundingClientRect() as DOMRect;
	await expect.element(screen.getByText("朝会", { exact: true })).toBeVisible();
	const between = rowOf("設計").top - rowOf("調査").bottom;
	expect(between).toBeGreaterThan(0);
	expect(rowOf("朝会").top - rowOf("設計").bottom).toBe(between);
});

test("記録がなければ、予定を一覧の上端から出す", async () => {
	const day = {
		...makeDay("2026-11-05"),
		plans: [{ title: "朝会", project: null, times: [], untimed: false }],
	};
	const screen = await renderApp(
		<WithDay value={dayValue(day)}>
			<DayPane />
		</WithDay>,
	);
	await expect.element(screen.getByText("朝会", { exact: true })).toBeVisible();
	const plans = document.querySelector(".rows.plans") as HTMLElement;
	expect(getComputedStyle(plans).marginTop).toBe("0px");
});

test("いつもの時間を時間不定にしたタスクの予定は、記録の行と同じく「不定」と出す", async () => {
	const day = {
		...makeDay("2026-11-05"),
		plans: [{ title: "情報収集", project: null, times: [], untimed: true }],
	};
	const screen = await renderApp(
		<WithDay value={dayValue(day)}>
			<DayPane />
		</WithDay>,
	);
	await expect.element(screen.getByText("不定", { exact: true })).toBeVisible();
});

import type { Repeat } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { formatDate } from "./date";

// 繰り返しの値は items.yaml と同じ書き方 (mo、2mo、-1fr など) で Go と受け渡す。
// 画面でするのは、選択肢との行き来と、一覧に出す短い文を作ることだけにする。

export const weekdays = [
	{ code: "mo", label: "月" },
	{ code: "tu", label: "火" },
	{ code: "we", label: "水" },
	{ code: "th", label: "木" },
	{ code: "fr", label: "金" },
	{ code: "sa", label: "土" },
	{ code: "su", label: "日" },
];

const weekdayLabel = (code: string) =>
	weekdays.find((w) => w.code === code)?.label ?? code;

const businessDays = ["mo", "tu", "we", "th", "fr"];

export type RepeatKind =
	| "none"
	| "daily"
	| "weekdays"
	| "weekly"
	| "biweekly"
	| "monthly";

export const repeatKinds: { value: RepeatKind; label: string }[] = [
	{ value: "none", label: "なし" },
	{ value: "daily", label: "毎日" },
	{ value: "weekdays", label: "平日" },
	{ value: "weekly", label: "毎週" },
	{ value: "biweekly", label: "隔週" },
	{ value: "monthly", label: "毎月" },
];

// RepeatForm は、ダイアログで選んでいる繰り返し。
// 選択肢を切り替えても、前に選んだ曜日などを覚えておけるよう、どの選択肢の値も持つ。
export type RepeatForm = {
	kind: RepeatKind;
	// days は毎週の曜日。隔週では先頭の 1 つを使う。
	days: string[];
	// monthly は、第 n 曜日 (nth) か、n 日 (day) か。
	monthly: "nth" | "day";
	// nth は週の番号 (1〜5、-1 は最終)。
	nth: string;
	nthDay: string;
	// monthDay は毎月の日。打ち直すあいだは空にできるよう、空文字も持つ。
	monthDay: number | "";
	start: string;
	until: string;
};

export function toForm(r: Repeat | null): RepeatForm {
	const form: RepeatForm = {
		kind: "none",
		days: [],
		monthly: "nth",
		nth: "1",
		nthDay: "mo",
		monthDay: 1,
		start: r?.start ?? "",
		until: r?.until ?? "",
	};
	if (!r) return form;
	const byDay = r.byDay ?? [];
	switch (r.freq) {
		case "daily":
			form.kind = "daily";
			break;
		case "weekly":
			form.days = byDay;
			if (r.interval === 2) form.kind = "biweekly";
			else if (
				byDay.length === businessDays.length &&
				businessDays.every((d) => byDay.includes(d))
			)
				form.kind = "weekdays";
			else form.kind = "weekly";
			break;
		case "monthly": {
			form.kind = "monthly";
			const m = /^(-?\d)(\w\w)$/.exec(byDay[0] ?? "");
			if (m) {
				form.nth = m[1];
				form.nthDay = m[2];
			} else {
				form.monthly = "day";
				form.monthDay = r.byMonthDay;
			}
			break;
		}
	}
	return form;
}

export function fromForm(f: RepeatForm): Repeat | null {
	const base = { start: f.start, until: f.until, byMonthDay: 0 };
	switch (f.kind) {
		case "none":
			return null;
		case "daily":
			return { ...base, freq: "daily", interval: 1, byDay: [] };
		case "weekdays":
			return { ...base, freq: "weekly", interval: 1, byDay: businessDays };
		case "weekly":
			return { ...base, freq: "weekly", interval: 1, byDay: f.days };
		case "biweekly":
			return {
				...base,
				freq: "weekly",
				interval: 2,
				byDay: f.days.slice(0, 1),
			};
		case "monthly":
			return f.monthly === "nth"
				? { ...base, freq: "monthly", interval: 1, byDay: [f.nth + f.nthDay] }
				: {
						...base,
						freq: "monthly",
						interval: 1,
						byDay: [],
						byMonthDay: f.monthDay === "" ? 0 : f.monthDay,
					};
	}
}

// describe は、タスクの管理の一覧に出す「隔週 水」のような短い文。
export function describe(r: Repeat): string {
	const f = toForm(r);
	let text = "";
	switch (f.kind) {
		case "daily":
			text = "毎日";
			break;
		case "weekdays":
			text = "平日";
			break;
		case "weekly":
			text = `毎週 ${f.days.map(weekdayLabel).join("・")}`;
			break;
		case "biweekly":
			text = `隔週 ${f.days.map(weekdayLabel).join("・")}`;
			break;
		case "monthly":
			text =
				f.monthly === "nth"
					? `毎月 ${f.nth === "-1" ? "最終" : `第${f.nth}`}${weekdayLabel(f.nthDay)}曜`
					: `毎月 ${f.monthDay}日`;
			break;
	}
	if (f.start || f.until) {
		text += ` ${f.start ? formatDate(f.start) : ""}〜${f.until ? formatDate(f.until) : ""}`;
	}
	return text;
}

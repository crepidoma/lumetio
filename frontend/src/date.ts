// 日付は Go と YYYY-MM-DD の文字列で受け渡す。
// 画面でするのは、前後の日への移動と、表示の文字を作ることだけにする。

function toDate(date: string): Date {
	const [y, m, d] = date.split("-").map(Number);
	return new Date(y, m - 1, d);
}

function toIsoDate(d: Date): string {
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function addDays(date: string, days: number): string {
	const d = toDate(date);
	d.setDate(d.getDate() + days);
	return toIsoDate(d);
}

const dayLabel = new Intl.DateTimeFormat("ja-JP", {
	year: "numeric",
	month: "2-digit",
	day: "2-digit",
	weekday: "short",
});

// formatDay は、ヘッダーに出す「2026/11/02(月)」の形。
export function formatDay(date: string): string {
	return dayLabel.format(toDate(date));
}

const dateLabel = new Intl.DateTimeFormat("ja-JP", {
	year: "numeric",
	month: "2-digit",
	day: "2-digit",
});

// formatDate は、期間に出す「2026/11/04」の形。
export function formatDate(date: string): string {
	return dateLabel.format(toDate(date));
}

// カレンダーの見出しなどは、dayjs の書式を書かずに済むよう、文字を返す関数で渡す。
const monthLabel = new Intl.DateTimeFormat("ja-JP", {
	year: "numeric",
	month: "long",
});
const yearLabel = new Intl.DateTimeFormat("ja-JP", { year: "numeric" });
const monthName = new Intl.DateTimeFormat("ja-JP", { month: "short" });
const weekdayName = new Intl.DateTimeFormat("ja-JP", { weekday: "narrow" });

export const calendarLabels = {
	monthLabelFormat: (date: string) => monthLabel.format(toDate(date)),
	yearLabelFormat: (date: string) => yearLabel.format(toDate(date)),
	decadeLabelFormat: (start: string, end: string) =>
		`${yearLabel.format(toDate(start))} 〜 ${yearLabel.format(toDate(end))}`,
	monthsListFormat: (date: string) => monthName.format(toDate(date)),
	yearsListFormat: (date: string) => String(toDate(date).getFullYear()),
	weekdayFormat: (date: string) => weekdayName.format(toDate(date)),
};

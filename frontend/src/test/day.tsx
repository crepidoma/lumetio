import type { ReactNode } from "react";
import type {
	Day,
	Note,
} from "../../bindings/github.com/crepidoma/lumetio/internal/gui";
import { DayContext, type DayContextValue } from "../DayContext";

// makeDay は、記録のない日。メモ欄だけを渡す。
export function makeDay(date: string, notes: Note[] = []): Day {
	return {
		date,
		exists: true,
		records: [],
		plans: [],
		notes,
		total: "",
		warnings: [],
		notices: [],
	};
}

// textNote は、定義のあるテキストのメモ欄。
export function textNote(name: string, text: string): Note {
	return { name, type: "text", text, bool: false, defined: true };
}

// dayValue は、日次画面の部品に渡す DayContext の値。操作は、渡した Go の操作をそのまま呼ぶ。
export function dayValue(day: Day, patch: Partial<DayContextValue> = {}) {
	const value: DayContextValue = {
		date: day.date,
		day,
		projects: [],
		tags: [],
		act: async (op) => {
			await op();
			return null;
		},
		run: async (op) => {
			await op();
		},
		addTask: async () => null,
		focus: null,
		reload: () => {},
		...patch,
	};
	return value;
}

export function WithDay({
	value,
	children,
}: {
	value: DayContextValue;
	children: ReactNode;
}) {
	return <DayContext value={value}>{children}</DayContext>;
}

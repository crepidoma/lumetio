import { createContext, useContext } from "react";
import type {
	Day,
	Opened,
	Project,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";

export type DayContextValue = {
	date: string;
	day: Day;
	projects: Project[];
	// act は、その日を書き換える操作を呼び、書いた後のその日を画面に出す。
	// 入力の誤りがあれば、その文を返す。
	act: (op: () => Promise<Opened>) => Promise<string | null>;
	// run は、入力欄のない操作 (ボタンやスイッチ) から act を呼ぶ。入力の誤りは、出す欄がないので知らせで出す。
	run: (op: () => Promise<Opened>) => Promise<void>;
	// addTask は、＋や検索からタスクをその日の記録に足し、その記録の時間を足す欄に移る。
	addTask: (title: string) => Promise<string | null>;
	// focus は、時間を足す欄に移る記録。seq が変わるたびに移る。
	focus: { title: string; seq: number } | null;
	// reload は、その日を読み直す。設定 (メモ欄の定義など) を変えた後に使う。
	reload: () => void;
};

export const DayContext = createContext<DayContextValue | null>(null);

export function useDay(): DayContextValue {
	const value = useContext(DayContext);
	if (!value) {
		throw new Error("DayContext の外で使われた");
	}
	return value;
}

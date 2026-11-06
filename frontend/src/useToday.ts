import { useEffect, useState } from "react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call } from "./api";

// 日付が変わってから「今日」が新しい日を指すまでの、いちばん長い遅れ。
const checkInterval = 60_000;

// useToday は今日の日付 (YYYY-MM-DD) を返す。読めるまでは null。
// 起動したまま日付が変わっても新しい日を返すよう、決まった間隔で読み直す。
// 日付が変わる時刻に合わせて 1 回だけ読む形にしないのは、PC が止まっていたあいだにその時刻を過ぎても、復帰したら追いつけるようにするため。
export function useToday() {
	const [today, setToday] = useState<string | null>(null);
	useEffect(() => {
		call(
			() => Service.Today(),
			() => {},
		).then((r) => {
			if (r.ok) setToday(r.value);
		});
		// 読み直しで失敗しても知らせない。毎回知らせると、同じ知らせが積もるため。前の日付のまま、次の読み直しを待つ。
		const id = setInterval(() => {
			Service.Today().then(setToday, () => {});
		}, checkInterval);
		return () => clearInterval(id);
	}, []);
	return today;
}

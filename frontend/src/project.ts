import type {
	Project,
	ProjectRow,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import type { Row } from "./SettingsSection";
import { sortedBy } from "./Sortable";

// projectLabel は、プロジェクトを選ぶ欄に出す名前。名前がなければコードだけを出す。
export function projectLabel(p: Project): string {
	return p.name && p.name !== p.code ? `${p.name} (${p.code})` : p.code;
}

// reorderShown は、出している行 (order) を、その並びに並べ替える。隠している期限切れの行は元の位置に残す。
// 期限切れは末尾にまとめて出すので、期限切れとそうでない行は、それぞれの中だけで並べ替える。
// 隠した行や、もう一方の組の行と位置を入れ替えると、出したときに思わぬ位置に動いて見えるため。
export function reorderShown(
	items: Row<ProjectRow>[],
	order: string[],
): Row<ProjectRow>[] {
	const moved = sortedBy(
		items.filter((i) => order.includes(i.id)),
		order,
		(i) => i.id,
	);
	const next = {
		active: moved.filter((i) => !i.row.expired),
		expired: moved.filter((i) => i.row.expired),
	};
	return items.map((i) => {
		if (!order.includes(i.id)) return i;
		return (i.row.expired ? next.expired : next.active).shift() ?? i;
	});
}

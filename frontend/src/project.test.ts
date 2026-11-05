import { expect, test } from "vitest";
import { reorderShown } from "./project";

function row(id: string, expired = false) {
	return { id, row: { code: id, name: "", start: "", until: "", expired } };
}

const ids = (rows: { id: string }[]) => rows.map((r) => r.id);

test("隠している期限切れの行は、元の位置に残す", () => {
	const items = [row("a"), row("x", true), row("b"), row("c")];
	expect(ids(reorderShown(items, ["c", "a", "b"]))).toEqual([
		"c",
		"x",
		"a",
		"b",
	]);
});

test("期限切れとそうでない行は、それぞれの中だけで並べ替える", () => {
	const items = [row("a"), row("b"), row("x", true), row("y", true)];
	expect(ids(reorderShown(items, ["b", "a", "y", "x"]))).toEqual([
		"b",
		"a",
		"y",
		"x",
	]);
	// 期限切れの行を、そうでない行のあいだへ動かしても、組をまたいでは動かさない。
	expect(ids(reorderShown(items, ["a", "x", "b", "y"]))).toEqual([
		"a",
		"b",
		"x",
		"y",
	]);
});

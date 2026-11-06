import { expect, test, vi } from "vitest";
import { TaskEditor } from "./TaskEditor";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: {},
}));

test("タグの欄は、タグの一覧を候補に出す", async () => {
	const screen = await renderApp(
		<TaskEditor
			date="2026-11-05"
			projects={[]}
			tagList={["ふりかえり", "調査"]}
			target=""
			onClose={() => {}}
			onChanged={() => {}}
			reload={() => {}}
		/>,
	);
	await screen.getByRole("combobox", { name: "タグ" }).click();
	await expect
		.element(screen.getByRole("option", { name: "ふりかえり" }))
		.toBeVisible();
	await expect
		.element(screen.getByRole("option", { name: "調査" }))
		.toBeVisible();
});

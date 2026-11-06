import { expect, test, vi } from "vitest";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { TaskEditor } from "./TaskEditor";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: {
		SaveTask: vi.fn(async () => {}),
		PreviewUsualTime: vi.fn(async () => []),
	},
}));

function renderEditor() {
	return renderApp(
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
}

test("タグの欄は、タグの一覧を候補に出す", async () => {
	const screen = await renderEditor();
	await screen.getByRole("combobox", { name: "タグ" }).click();
	await expect
		.element(screen.getByRole("option", { name: "ふりかえり" }))
		.toBeVisible();
	await expect
		.element(screen.getByRole("option", { name: "調査" }))
		.toBeVisible();
});

test("いつもの時間を時間不定にすることは、それだけで設定になり、保存できる", async () => {
	const screen = await renderEditor();
	await screen.getByRole("textbox", { name: "タイトル" }).fill("情報収集");
	const save = screen.getByRole("button", { name: "保存" });
	await expect.element(save).toBeDisabled();
	await screen.getByRole("checkbox", { name: "時間不定" }).click();
	await save.click();
	await vi.waitFor(() =>
		expect(Service.SaveTask).toHaveBeenCalledExactlyOnceWith(
			"",
			null,
			expect.objectContaining({ title: "情報収集", time: "", untimed: true }),
		),
	);
});

test("いつもの時間と時間不定は、片方が入っているあいだ、もう片方を使えない", async () => {
	const screen = await renderEditor();
	const usual = screen.getByRole("textbox", { name: "いつもの時間" });
	const untimed = screen.getByRole("checkbox", { name: "時間不定" });
	await usual.fill("30m");
	await expect.element(untimed).toBeDisabled();
	await usual.fill("");
	await untimed.click();
	await expect.element(usual).toBeDisabled();
});

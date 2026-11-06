import { beforeEach, expect, test, vi } from "vitest";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { TagsSection } from "./TagsSection";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { TagRows: vi.fn(), SaveTags: vi.fn(async () => {}) },
}));

beforeEach(() => {
	vi.mocked(Service.TagRows).mockResolvedValue(["ふりかえり"]);
	vi.mocked(Service.SaveTags).mockClear();
});

test("名前を直し、タグを足して保存すると、読んだときの一覧と一緒に書く", async () => {
	const onSaved = vi.fn();
	const screen = await renderApp(
		<TagsSection onDirty={() => {}} onSaved={onSaved} reload={() => {}} />,
	);
	await screen.getByRole("textbox", { name: "名前" }).fill("振り返り");
	await screen.getByRole("button", { name: "タグを足す" }).click();
	await screen.getByRole("textbox", { name: "名前" }).nth(1).fill("調査");
	await screen.getByRole("button", { name: "保存" }).click();
	await vi.waitFor(() =>
		expect(Service.SaveTags).toHaveBeenCalledExactlyOnceWith(
			["ふりかえり"],
			["振り返り", "調査"],
		),
	);
	await vi.waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
});

test("変えていなければ保存できない", async () => {
	const screen = await renderApp(
		<TagsSection onDirty={() => {}} onSaved={() => {}} reload={() => {}} />,
	);
	await expect
		.element(screen.getByRole("textbox", { name: "名前" }))
		.toHaveValue("ふりかえり");
	await expect
		.element(screen.getByRole("button", { name: "保存" }))
		.toBeDisabled();
});

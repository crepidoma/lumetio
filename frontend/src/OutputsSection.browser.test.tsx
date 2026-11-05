import { expect, test, vi } from "vitest";
import {
	Service,
	type TemplateDraft,
	type TemplateEntry,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { OutputsSection } from "./OutputsSection";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { Templates: vi.fn(), PreviewTemplate: vi.fn() },
}));

function textEntry(name: string): TemplateEntry {
	return {
		fileName: `${name}.tmpl`,
		name,
		kind: "text",
		data: name,
		error: "",
		text: {
			file: "'report_'yyyyMMdd",
			extension: "txt",
			encoding: "utf-8",
			newline: "lf",
			body: name,
		},
		csv: null,
		set: null,
	};
}

test("出力を選び直したら、前の出力のプレビューを残さない", async () => {
	vi.mocked(Service.Templates).mockResolvedValue({
		entries: [textEntry("日報"), textEntry("週報")],
		default: "",
		notices: [],
	});
	// 週報の結果は届かないままにし、そのあいだに出ているものを見る。
	vi.mocked(Service.PreviewTemplate).mockImplementation(
		(_date: string, d: TemplateDraft) =>
			(d.name === "日報"
				? Promise.resolve({
						name: "日報",
						fileName: "report_20261105.txt",
						text: "日報の結果",
						error: "",
					})
				: new Promise(() => {})) as ReturnType<typeof Service.PreviewTemplate>,
	);
	const screen = await renderApp(
		<OutputsSection
			date="2026-11-05"
			onDirty={() => {}}
			onSaved={() => {}}
			reload={() => {}}
		/>,
	);
	await screen.getByRole("button", { name: /^日報/ }).click();
	const old = screen.getByText("日報の結果");
	await expect.element(old).toBeVisible();
	await screen.getByRole("button", { name: /^週報/ }).click();
	await expect.element(screen.getByLabelText("名前")).toHaveValue("週報");
	await expect.element(old).not.toBeInTheDocument();
});

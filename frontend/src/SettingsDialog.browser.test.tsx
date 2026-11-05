import { beforeEach, expect, test, vi } from "vitest";
import { userEvent } from "vitest/browser";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { SettingsDialog } from "./SettingsDialog";
import { inputError, renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: {
		Locations: vi.fn(),
		DefaultLocations: vi.fn(),
		AtDefault: vi.fn(),
		DurationFormat: vi.fn(),
		DefaultDurationFormat: vi.fn(),
		PreviewDurationFormat: vi.fn(),
		SaveDurationFormat: vi.fn(),
		Templates: vi.fn(),
		ProjectRows: vi.fn(),
		ChooseCSV: vi.fn(),
		ReadCSV: vi.fn(),
	},
}));

const dirs = { dataDir: "D:/data", outputDir: "D:/out" };

beforeEach(() => {
	vi.mocked(Service.Locations).mockResolvedValue(dirs);
	vi.mocked(Service.DefaultLocations).mockResolvedValue(dirs);
	vi.mocked(Service.AtDefault).mockResolvedValue({
		dataDir: true,
		outputDir: true,
	});
	vi.mocked(Service.DurationFormat).mockResolvedValue("h:mm");
	vi.mocked(Service.DefaultDurationFormat).mockResolvedValue("h:mm");
	vi.mocked(Service.PreviewDurationFormat).mockResolvedValue([]);
	vi.mocked(Service.SaveDurationFormat).mockReset();
	vi.mocked(Service.ProjectRows).mockResolvedValue([]);
	vi.mocked(Service.ChooseCSV).mockResolvedValue("D:/projects.csv");
	vi.mocked(Service.ReadCSV).mockResolvedValue([
		["code", "name"],
		["P-1", "試作"],
	]);
	vi.mocked(Service.Templates).mockResolvedValue({
		entries: [
			{
				fileName: "壊れた.yaml",
				name: "",
				kind: "text",
				data: "",
				error: "読めません",
				text: null,
				csv: null,
				set: null,
			},
		],
		default: "",
		notices: [],
	});
});

test("項目の中の確かめるダイアログを Esc で閉じても、設定は閉じない", async () => {
	const onClose = vi.fn();
	const screen = await renderApp(
		<SettingsDialog
			date="2026-11-05"
			opened
			onClose={onClose}
			onLocationsChanged={() => {}}
			onOutputsChanged={() => {}}
			reload={() => {}}
		/>,
	);
	await screen.getByRole("tab", { name: "出力" }).click();
	await screen.getByRole("button", { name: /^壊れた.yaml/ }).click();
	await screen.getByRole("button", { name: "削除", exact: true }).click();
	const confirm = screen.getByText("を削除しますか？");
	await expect.element(confirm).toBeVisible();
	await userEvent.keyboard("{Escape}");
	await expect.element(confirm).not.toBeInTheDocument();
	expect(onClose).not.toHaveBeenCalled();
	await expect.element(screen.getByRole("tab", { name: "出力" })).toBeVisible();
});

async function openDuration(
	props: { onClose?: () => void; reload?: () => void } = {},
) {
	const screen = await renderApp(
		<SettingsDialog
			date="2026-11-05"
			opened
			onClose={props.onClose ?? (() => {})}
			onLocationsChanged={() => {}}
			onOutputsChanged={() => {}}
			reload={props.reload ?? (() => {})}
		/>,
	);
	await screen.getByRole("tab", { name: "長さの表記" }).click();
	const field = screen.getByRole("textbox", { name: "長さの表記" });
	await expect.element(field).toHaveValue("h:mm");
	return { screen, field };
}

test("保存していない変更があるまま項目を移るときは確かめ、破棄したら保存した値から始める", async () => {
	const { screen, field } = await openDuration();
	await field.fill("hh:mm");
	await screen.getByRole("tab", { name: "場所" }).click();
	await screen.getByRole("button", { name: "破棄する" }).click();
	await expect
		.element(screen.getByRole("tab", { name: "場所" }))
		.toHaveAttribute("aria-selected", "true");

	// 戻ると、打った値ではなく保存した値から始まり、移るときにもう確かめない。
	await screen.getByRole("tab", { name: "長さの表記" }).click();
	await expect.element(field).toHaveValue("h:mm");
	await screen.getByRole("tab", { name: "場所" }).click();
	await expect
		.element(screen.getByRole("tab", { name: "場所" }))
		.toHaveAttribute("aria-selected", "true");
});

test("確かめてキャンセルしたら、その項目に変更を残したままにする", async () => {
	const { screen, field } = await openDuration();
	await field.fill("hh:mm");
	await screen.getByRole("tab", { name: "場所" }).click();
	await screen.getByRole("button", { name: "キャンセル" }).click();
	await expect.element(field).toHaveValue("hh:mm");
	await expect
		.element(screen.getByRole("tab", { name: "長さの表記" }))
		.toHaveAttribute("aria-selected", "true");
});

test("保存していない変更があるまま閉じるときも確かめる", async () => {
	const onClose = vi.fn();
	const { screen, field } = await openDuration({ onClose });
	await field.fill("hh:mm");
	await userEvent.keyboard("{Escape}");
	await expect
		.element(screen.getByText("保存していない変更を破棄しますか？"))
		.toBeVisible();
	expect(onClose).not.toHaveBeenCalled();
	await screen.getByRole("button", { name: "破棄する" }).click();
	expect(onClose).toHaveBeenCalledOnce();
});

test("保存の誤りはボタンの横に出し、入力を直したら消す", async () => {
	vi.mocked(Service.SaveDurationFormat).mockRejectedValue(
		inputError("保存の誤り"),
	);
	const { screen, field } = await openDuration();
	await field.fill("hh:mm");
	await screen.getByRole("button", { name: "保存" }).click();
	const message = screen.getByText("保存の誤り");
	await expect.element(message).toBeVisible();
	await field.fill("hh:mm:ss");
	await expect.element(message).not.toBeInTheDocument();
});

test("保存の誤りが欄の誤りと同じなら、ボタンの横に重ねて出さない", async () => {
	vi.mocked(Service.PreviewDurationFormat).mockRejectedValue(
		new Error("読めない表記"),
	);
	vi.mocked(Service.SaveDurationFormat).mockRejectedValue(
		inputError("読めない表記"),
	);
	const { screen, field } = await openDuration();
	await field.fill("x");
	await expect.element(screen.getByText("読めない表記")).toBeVisible();
	await screen.getByRole("button", { name: "保存" }).click();
	await vi.waitFor(() => expect(Service.SaveDurationFormat).toHaveBeenCalled());
	// 保存の結果を画面に出し終えるのを待ってから数える。
	await new Promise((r) => setTimeout(r, 100));
	expect(screen.getByText("読めない表記").elements()).toHaveLength(1);
});

test("保存が入力の誤りでも衝突でもなく失敗したら、読み直す", async () => {
	vi.mocked(Service.SaveDurationFormat).mockRejectedValue(
		new Error("書けません"),
	);
	const reload = vi.fn();
	const { screen, field } = await openDuration({ reload });
	await field.fill("hh:mm");
	await screen.getByRole("button", { name: "保存" }).click();
	await vi.waitFor(() => expect(reload).toHaveBeenCalled());
});

async function openImport(onClose: () => void = () => {}) {
	const screen = await renderApp(
		<SettingsDialog
			date="2026-11-05"
			opened
			onClose={onClose}
			onLocationsChanged={() => {}}
			onOutputsChanged={() => {}}
			reload={() => {}}
		/>,
	);
	await screen.getByRole("tab", { name: "プロジェクト" }).click();
	await screen.getByRole("button", { name: "CSV から取り込む" }).click();
	const dialog = screen.getByRole("dialog", { name: "CSV から取り込む" });
	await expect.element(dialog).toBeVisible();
	return { screen, dialog };
}

test("CSV から取り込むダイアログは、設定の上に出る", async () => {
	const { dialog } = await openImport();
	const box = dialog.element().getBoundingClientRect();
	const top = document.elementFromPoint(
		box.left + box.width / 2,
		box.top + box.height / 2,
	);
	expect(dialog.element().contains(top)).toBe(true);
});

test("CSV から取り込むダイアログを開いている間も、設定は後ろに見えている", async () => {
	const { screen } = await openImport();
	const settings = screen.getByRole("dialog", { name: "設定" }).element();
	// Mantine は重ねた下のダイアログを透明にして隠すので、見えているかは不透明度で見る。
	// 透明になるまでの動きが終わってから測る。
	await new Promise((r) => requestAnimationFrame(r));
	await Promise.all(settings.getAnimations().map((a) => a.finished));
	expect(getComputedStyle(settings).opacity).toBe("1");
});

test("CSV から取り込むダイアログを開いている間も、設定の暗い幕を残し、本体と見分けられる", async () => {
	await openImport();
	// 設定の幕と取り込みの幕の 2 枚。本体は 2 枚分、設定は取り込みの 1 枚分だけ暗くなる。
	await vi.waitFor(() =>
		expect(document.querySelectorAll(".mantine-Modal-overlay")).toHaveLength(2),
	);
});

// 設定のダイアログが、上に重ねたダイアログに隠されたままになっていないか。
async function expectSettingsShown(
	screen: Awaited<ReturnType<typeof openImport>>["screen"],
) {
	const settings = screen.getByRole("dialog", { name: "設定" });
	await expect.element(settings).toBeVisible();
	await vi.waitFor(() =>
		expect(settings.element().closest("[data-hidden]")).toBeNull(),
	);
	await expect
		.element(screen.getByRole("button", { name: "CSV から取り込む" }))
		.toBeVisible();
}

test("CSV から取り込むダイアログを Esc で閉じても、設定は閉じない", async () => {
	const onClose = vi.fn();
	const { screen, dialog } = await openImport(onClose);
	await userEvent.keyboard("{Escape}");
	await expect.element(dialog).not.toBeInTheDocument();
	expect(onClose).not.toHaveBeenCalled();
	await expectSettingsShown(screen);
});

test("CSV から取り込むダイアログをキャンセルしたら、設定に戻り、もう一度取り込める", async () => {
	const { screen, dialog } = await openImport();
	await screen.getByRole("button", { name: "キャンセル" }).click();
	await expect.element(dialog).not.toBeInTheDocument();
	await expectSettingsShown(screen);
	await screen.getByRole("button", { name: "CSV から取り込む" }).click();
	await expect.element(dialog).toBeVisible();
});

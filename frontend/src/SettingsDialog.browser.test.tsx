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

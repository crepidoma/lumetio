import { expect, test, vi } from "vitest";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { NoteDefsDialog } from "./NoteDefsSection";
import { escapeToConfirm } from "./test/escape";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { NoteDefRows: vi.fn() },
}));

test("保存していない変更があるまま Esc で閉じるときは確かめる", async () => {
	vi.mocked(Service.NoteDefRows).mockResolvedValue([
		{ name: "連絡", type: "text", carryOver: false },
	]);
	const onClose = vi.fn();
	const screen = await renderApp(
		<NoteDefsDialog opened onClose={onClose} reload={() => {}} />,
	);
	await screen.getByRole("textbox", { name: "名前" }).fill("共有");
	await escapeToConfirm(
		screen.getByText("保存していない変更を破棄しますか？"),
		onClose,
	);
	await screen.getByRole("button", { name: "破棄する" }).click();
	expect(onClose).toHaveBeenCalledOnce();
});

import { beforeEach, expect, test, vi } from "vitest";
import { userEvent } from "vitest/browser";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { NotesSection } from "./NotesSection";
import { dayValue, makeDay, textNote, WithDay } from "./test/day";
import { renderApp } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { SetText: vi.fn(async () => ({})) },
}));

beforeEach(() => {
	vi.mocked(Service.SetText).mockClear();
});

function notes(date: string, text: string) {
	return (
		<WithDay value={dayValue(makeDay(date, [textNote("連絡", text)]))}>
			<NotesSection />
		</WithDay>
	);
}

test("書くときに、打ち始めたときの値も渡す", async () => {
	const screen = await renderApp(notes("2026-11-05", "前"));
	await screen.getByRole("textbox", { name: "連絡" }).fill("後");
	await screen.getByRole("heading", { name: "メモ欄" }).click();
	expect(Service.SetText).toHaveBeenCalledExactlyOnceWith(
		"2026-11-05",
		"連絡",
		"前",
		"後",
	);
});

test("日を移ったら、前の日の打ちかけの内容を移った日に書かない", async () => {
	const screen = await renderApp(notes("2026-11-05", ""));
	const field = screen.getByRole("textbox", { name: "連絡" });
	await field.fill("打ちかけ");
	await screen.rerender(notes("2026-11-06", ""));
	await expect.element(field).toHaveValue("");
	await screen.getByRole("heading", { name: "メモ欄" }).click();
	expect(Service.SetText).not.toHaveBeenCalled();
});

// 書けなかった理由を返す act。画面では入力の誤りの文を返す。
const failing = dayValue(makeDay("2026-11-05", [textNote("連絡", "")]), {
	act: async () => "書けない値です",
});

test("書けなかった吹き出しは、打ち直したら消す", async () => {
	const screen = await renderApp(
		<WithDay value={failing}>
			<NotesSection />
		</WithDay>,
	);
	const field = screen.getByRole("textbox", { name: "連絡" });
	await field.fill("値");
	await screen.getByRole("heading", { name: "メモ欄" }).click();
	const message = screen.getByText("書けない値です");
	await expect.element(message).toBeVisible();
	await field.fill("値を直す");
	await expect.element(message).not.toBeInTheDocument();
});

test("書けなかった吹き出しは、Esc で消す", async () => {
	const screen = await renderApp(
		<WithDay value={failing}>
			<NotesSection />
		</WithDay>,
	);
	const field = screen.getByRole("textbox", { name: "連絡" });
	await field.fill("値");
	await screen.getByRole("heading", { name: "メモ欄" }).click();
	const message = screen.getByText("書けない値です");
	await expect.element(message).toBeVisible();
	await field.click();
	await userEvent.keyboard("{Escape}");
	await expect.element(message).not.toBeInTheDocument();
});

test("打っているあいだに他で書き換わったら、その値にする", async () => {
	const screen = await renderApp(notes("2026-11-05", ""));
	const field = screen.getByRole("textbox", { name: "連絡" });
	await field.fill("打ちかけ");
	await screen.rerender(notes("2026-11-05", "他の PC"));
	await expect.element(field).toHaveValue("他の PC");
	await screen.getByRole("heading", { name: "メモ欄" }).click();
	expect(Service.SetText).not.toHaveBeenCalled();
});

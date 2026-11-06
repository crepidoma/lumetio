import type { ReactNode } from "react";
import { expect, test, vi } from "vitest";
import { render, renderHook } from "vitest-browser-react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useOutputs } from "./OutputPane";
import { dayValue, makeDay, WithDay } from "./test/day";
import { Providers } from "./test/render";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { OutputChoices: vi.fn(), Preview: vi.fn() },
}));

const day = dayValue(makeDay("2026-11-05"));

function wrapper({ children }: { children: ReactNode }) {
	return (
		<Providers>
			<WithDay value={day}>{children}</WithDay>
		</Providers>
	);
}

const noNotices = () => {};

test("出力の一覧を読めなくなったら、前のプレビューをコピーできるまま残さない", async () => {
	vi.mocked(Service.OutputChoices).mockResolvedValue({
		sets: [],
		outputs: ["日報"],
		default: "日報",
		notices: [],
	});
	vi.mocked(Service.Preview).mockResolvedValue({
		outputs: [{ name: "日報", fileName: "日報.txt", text: "本文", error: "" }],
		notices: [],
	});
	const { result, rerender } = await renderHook(
		(version = 0) => useOutputs(noNotices, version),
		{ initialProps: 0, wrapper },
	);
	await vi.waitFor(() => expect(result.current.current?.text).toBe("本文"));

	vi.mocked(Service.OutputChoices).mockRejectedValue(new Error("読めません"));
	await rerender(1);
	await vi.waitFor(() => expect(result.current.choices).toBeNull());
	expect(result.current.current).toBeUndefined();
});

test("選び直したら、前に選んでいたもののプレビューを残さない", async () => {
	vi.mocked(Service.OutputChoices).mockResolvedValue({
		sets: [],
		outputs: ["日報", "工数"],
		default: "日報",
		notices: [],
	});
	// 工数の結果は届かないままにし、そのあいだに出ているものを見る。
	vi.mocked(Service.Preview).mockImplementation(
		(_date: string, name: string) =>
			(name === "日報"
				? Promise.resolve({
						outputs: [
							{ name: "日報", fileName: "日報.txt", text: "本文", error: "" },
						],
						notices: [],
					})
				: new Promise(() => {})) as ReturnType<typeof Service.Preview>,
	);
	const { result, act } = await renderHook(() => useOutputs(noNotices, 0), {
		wrapper,
	});
	await vi.waitFor(() => expect(result.current.current?.text).toBe("本文"));

	await act(() => result.current.select("工数"));
	expect(result.current.name).toBe("工数");
	expect(result.current.current).toBeUndefined();
});

// Pane は、コピーのボタンと確かめるダイアログを出す。useOutputs はダイアログを返すだけなので、描いて確かめるため。
function Pane() {
	const out = useOutputs(noNotices, 0);
	return (
		<>
			<button type="button" onClick={out.copy}>
				コピー
			</button>
			<span>{out.current?.text}</span>
			{out.dialogs}
		</>
	);
}

// 計上する記録の時間未入力は、期間の外のプロジェクトと同じく、アプリの外へ出す前に確かめる。工数を取り込む前に、入れ忘れに気づけるようにするため。
test("時間未入力の警告があれば、コピーの前に確かめる", async () => {
	vi.mocked(Service.OutputChoices).mockResolvedValue({
		sets: [],
		outputs: ["日報"],
		default: "日報",
		notices: [],
	});
	vi.mocked(Service.Preview).mockResolvedValue({
		outputs: [{ name: "日報", fileName: "日報.txt", text: "本文", error: "" }],
		notices: [],
	});
	const warned = dayValue({
		...makeDay("2026-11-05"),
		warnings: ["調査 の時間が入っていません"],
	});
	const screen = await render(<Pane />, {
		wrapper: ({ children }) => (
			<Providers>
				<WithDay value={warned}>{children}</WithDay>
			</Providers>
		),
	});
	await expect.element(screen.getByText("本文")).toBeVisible();
	await screen.getByRole("button", { name: "コピー" }).click();
	await expect.element(screen.getByText("続けますか？")).toBeVisible();
	await expect
		.element(screen.getByText("調査 の時間が入っていません"))
		.toBeVisible();
});

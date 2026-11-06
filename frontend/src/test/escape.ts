import type { Mock } from "vitest";
import { expect, vi } from "vitest";
import { type Locator, userEvent } from "vitest/browser";

// 押し直す回数の上限。届くのを待つだけなので、数回で足りる。
const maxTries = 5;

// escapeToConfirm は、保存していない変更があるダイアログで Esc を押し、確かめるダイアログ (confirm) が出るまで待つ。
// 打った内容は effect を通ってダイアログに届くので、打った直後に押した Esc は、変更がないものとして閉じる (onClose を呼ぶ) ことがある。
// そのときは押し直す。押すたびに、確かめるか閉じるかのどちらかが起きるのを待つので、出た確認を次の Esc で閉じることはない。
// 押し直す前に呼ばれた onClose は数えないよう、消しておく。
export async function escapeToConfirm(confirm: Locator, onClose: Mock) {
	for (let i = 0; i < maxTries; i++) {
		onClose.mockClear();
		await userEvent.keyboard("{Escape}");
		await vi.waitFor(() =>
			expect(confirm.query() !== null || onClose.mock.calls.length > 0).toBe(
				true,
			),
		);
		if (confirm.query() !== null) {
			await expect.element(confirm).toBeVisible();
			expect(onClose).not.toHaveBeenCalled();
			return;
		}
	}
	throw new Error(`Esc を ${maxTries} 回押しても、確かめるダイアログが出ない`);
}

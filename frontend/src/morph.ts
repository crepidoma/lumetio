import { flushSync } from "react-dom";

// morph は、画面の切り替えをビュー遷移 (View Transitions) で行う。
// 同じ view-transition-name を持つ要素は、切り替えの前の位置と大きさから、後の位置と大きさへ移っていく。
// 別々の要素が消えて現れるのではなく、同じものが動いて見えるようにするため。
// ビュー遷移に対応していないときと、OS で「アニメーションを減らす」にしているときは、そのまま切り替える。
export function morph(update: () => void) {
	if (
		!document.startViewTransition ||
		matchMedia("(prefers-reduced-motion: reduce)").matches
	) {
		update();
		return;
	}
	// ビュー遷移は、切り替えた後の画面をすぐに写すので、React に変更をその場で反映させる。
	document.startViewTransition(() => flushSync(update));
}

// morphNames は、バーと右ペインのあいだで移っていくボタンに付ける名前。
// 同じ名前は画面に 1 つしか置けないので、そのとき見えている方だけに付ける。
export const morphNames = {
	copy: { viewTransitionName: "output-copy" },
	export: { viewTransitionName: "output-export" },
} as const;

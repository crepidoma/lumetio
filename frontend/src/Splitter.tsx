import { useRef } from "react";

type Props = {
	label: string;
	// value、min、max は、読み上げのために今の幅と範囲を伝える。
	value: number;
	min: number;
	max: number;
	onStart: () => void;
	// onMove は、ドラッグを始めた位置から右へ動かした量 (px) を渡す。
	onMove: (dx: number) => void;
	onEnd: () => void;
	// onKey は、左キーなら -1、右キーなら 1 を渡す。
	onKey: (dir: -1 | 1) => void;
};

// Splitter はペインの境目。ドラッグと左右キーで幅を変える。
export function Splitter({
	label,
	value,
	min,
	max,
	onStart,
	onMove,
	onEnd,
	onKey,
}: Props) {
	const startX = useRef<number | null>(null);
	return (
		// biome-ignore lint/a11y/useSemanticElements: 縦の境目には、ドラッグできる hr のような要素がないため、role で境目と伝える。
		<div
			role="separator"
			aria-orientation="vertical"
			aria-label={label}
			aria-valuenow={value}
			aria-valuemin={min}
			aria-valuemax={max}
			tabIndex={0}
			className="splitter"
			onPointerDown={(e) => {
				e.currentTarget.setPointerCapture(e.pointerId);
				startX.current = e.clientX;
				onStart();
			}}
			// 動かした量は整数に丸める。表示を拡大していると位置が小数になるが、幅は整数で保存するため。
			onPointerMove={(e) => {
				if (startX.current !== null)
					onMove(Math.round(e.clientX - startX.current));
			}}
			onPointerUp={(e) => {
				if (startX.current === null) return;
				e.currentTarget.releasePointerCapture(e.pointerId);
				startX.current = null;
				onEnd();
			}}
			onKeyDown={(e) => {
				if (e.key === "ArrowLeft") {
					e.preventDefault();
					onKey(-1);
				} else if (e.key === "ArrowRight") {
					e.preventDefault();
					onKey(1);
				}
			}}
		/>
	);
}

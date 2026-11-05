import { useViewportSize } from "@mantine/hooks";
import {
	type ReactNode,
	useCallback,
	useEffect,
	useRef,
	useState,
} from "react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call } from "./api";
import { useDay } from "./DayContext";
import { DayPane } from "./DayPane";
import { ChevronsRight } from "./icons";
import { morph } from "./morph";
import { NotesSection } from "./NotesSection";
import { OutputBar, OutputPane, type Outputs, useOutputs } from "./OutputPane";
import { Splitter } from "./Splitter";

// ペインの幅の決まり (px)。値は仮で、使いながら調整する。
const leftRange = { min: 200, max: 480, initial: 260 };
const rightRange = { min: 280, max: 640, initial: 380 };
// 中央はこれより狭くしない。足りなくなったら左を縮め、それでも足りなければ右を畳む。
const centerMin = 400;
// ウィンドウがこれより狭いと、右を畳む。
const wideFrom = 1100;
// 畳んだ右のバーと、境目の幅。CSS の .right-bar と .splitter に合わせる。
const barWidth = 72;
const splitterWidth = 6;
// 左右キーで動かす幅。
const keyStep = 16;
// 左右キーで動かした後、保存するまで待つ時間 (ms)。押し続けているあいだに何度も書かないため。
const keySaveDelay = 500;
// 狭いときに引き出す右ペインの幅と、ウィンドウの左端に残す隙間。
const sheetMaxWidth = 560;
const sheetMargin = 80;

const clamp = (v: number, min: number, max: number) =>
	Math.max(min, Math.min(v, max));

type Props = {
	left: ReactNode;
	onNotices: (messages: string[]) => void;
	// outputsVersion は、出力と出力セットが変わるたびに増える。右ペインの出力の一覧を読み直すため。
	outputsVersion: number;
};

// Workspace は、左、中央、右のペインとその境目。幅を変え、右を畳み、幅を保存する。
export function Workspace({
	left: leftPane,
	onNotices,
	outputsVersion,
}: Props) {
	const out = useOutputs(onNotices, outputsVersion);
	const { width: viewport } = useViewportSize();
	const [left, setLeft] = useState(leftRange.initial);
	// right は右ペインの幅。畳んでも覚えておき、開くと元の幅に戻す。
	const [right, setRight] = useState(rightRange.initial);
	// collapsed は、閉じるボタンで手で畳んだか。local.yaml に保存し、次に開いたときもウィンドウの幅をまたいでも覚えておく。
	const [collapsed, setCollapsed] = useState(false);
	// sheet は、狭くて自動で畳んでいるときに、右ペインをバーから引き出しているか。
	const [sheet, setSheetState] = useState(false);
	// dragging は、右の境目をドラッグしているか。そのあいだは幅をなめらかに変えない。
	const [dragging, setDragging] = useState(false);

	useEffect(() => {
		call(
			() => Service.Panes(),
			() => {},
		).then((r) => {
			if (!r.ok) return;
			// 保存されていない (0) か範囲の外なら、初期値か範囲に収める。手で書き換えた値でも開けるようにするため。
			setLeft(
				clamp(r.value.left || leftRange.initial, leftRange.min, leftRange.max),
			);
			setRight(
				clamp(
					r.value.right || rightRange.initial,
					rightRange.min,
					rightRange.max,
				),
			);
			setCollapsed(r.value.rightCollapsed);
		});
	}, []);

	// ウィンドウが狭いか、右ペインを置くと中央が最小より狭くなるときは、自動で畳む。
	const roomForRight =
		viewport - leftRange.min - centerMin - splitterWidth * 2 >= right;
	const auto = viewport < wideFrom || !roomForRight;
	const open = !auto && !collapsed;
	// 右の境目を動かすときは、中央が最小の幅になるところで止める。そこを越えると自動で畳まれ、ドラッグの途中で消えてしまうため。
	const rightMax = Math.max(
		rightRange.min,
		Math.min(
			rightRange.max,
			viewport - leftRange.min - centerMin - splitterWidth * 2,
		),
	);

	// 自動で畳むかどうかが変わったら、引き出した右ペインはしまう。手で畳んだかは覚えておく。
	const lastAuto = useRef(auto);
	useEffect(() => {
		if (lastAuto.current !== auto) {
			lastAuto.current = auto;
			setSheetState(false);
		}
	}, [auto]);

	const shownLeft = clamp(
		left,
		leftRange.min,
		Math.min(
			leftRange.max,
			viewport - centerMin - (open ? right : barWidth) - splitterWidth * 2,
		),
	);

	const save = useCallback((l: number, r: number, c: boolean) => {
		call(
			() => Service.SavePanes({ left: l, right: r, rightCollapsed: c }),
			() => {},
		);
	}, []);

	// 左右キーで動かしたときは、少し待ってから保存する。
	const keyTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
	const saveLater = (l: number, r: number) => {
		clearTimeout(keyTimer.current);
		keyTimer.current = setTimeout(() => save(l, r, collapsed), keySaveDelay);
	};

	// ドラッグを始めたときの幅。動かした量はそこからの差で決める。
	const dragStart = useRef(0);

	// 引き出すときは、中央を押し縮めないので、右ペインより広めにする。ウィンドウが狭くても左端に少し隙間を残す。
	const sheetWidth = Math.min(sheetMaxWidth, viewport - barWidth - sheetMargin);
	// 手で開け閉めしたら、すぐ保存する。狭いときに引き出したことは、その場限りなので保存しない。
	const setRightCollapsed = (c: boolean) => {
		setCollapsed(c);
		save(left, right, c);
	};
	// 引き出すとき、しまうときは、バーのコピーと出力が右ペインの中のコピーと出力へ移っていくように見せる。
	const setSheet = (o: boolean) => {
		if (o !== sheet) morph(() => setSheetState(o));
	};
	const openRight = () => (auto ? setSheet(true) : setRightCollapsed(false));
	const closeRight = () => (auto ? setSheet(false) : setRightCollapsed(true));
	const sheetOpen = auto && sheet;

	// 引き出した右ペインは、Esc と、右ペインとバーの外を押したときにも閉じる。押した操作 (日付を変える、＋など) はそのまま行う。
	// 右ペインやバーの中から開いた候補の一覧、ダイアログ、知らせは画面の別の場所 (data-portal) に出るので、外に数えない。
	// 数えると、出力を選ぶ一覧から選んだだけで閉じてしまうため。
	const sheetRef = useRef<HTMLElement>(null);
	const barRef = useRef<HTMLElement>(null);
	useEffect(() => {
		if (!sheetOpen) return;
		const close = () => morph(() => setSheetState(false));
		const onKey = (e: KeyboardEvent) => {
			if (e.key !== "Escape") return;
			// Esc は、いちばん上のものだけを閉じる。確認のダイアログや、開いている候補の一覧があれば、そちらに任せる。
			if (document.querySelector('[role="dialog"][aria-modal="true"]')) return;
			if (document.activeElement?.getAttribute("aria-expanded") === "true")
				return;
			close();
		};
		const onPointer = (e: PointerEvent) => {
			const target = e.target as Element;
			if (
				sheetRef.current?.contains(target) ||
				barRef.current?.contains(target)
			)
				return;
			if (target.closest("[data-portal]")) return;
			close();
		};
		window.addEventListener("keydown", onKey);
		window.addEventListener("pointerdown", onPointer);
		return () => {
			window.removeEventListener("keydown", onKey);
			window.removeEventListener("pointerdown", onPointer);
		};
	}, [sheetOpen]);

	// ＋や検索で足した記録の時間の欄に移るときは、引き出した右ペインを閉じる。重ねた右ペインの下に隠れた欄に打たないようにするため。
	const { focus } = useDay();
	const sheetNow = useRef(sheet);
	sheetNow.current = sheet;
	useEffect(() => {
		if (focus && sheetNow.current) morph(() => setSheetState(false));
	}, [focus]);

	return (
		<div className="body">
			<aside className="pane left" style={{ width: shownLeft }}>
				{leftPane}
			</aside>
			<Splitter
				label="左ペインの幅"
				value={shownLeft}
				min={leftRange.min}
				max={leftRange.max}
				onStart={() => {
					dragStart.current = shownLeft;
				}}
				onMove={(dx) =>
					setLeft(clamp(dragStart.current + dx, leftRange.min, leftRange.max))
				}
				onEnd={() => save(left, right, collapsed)}
				onKey={(dir) => {
					const next = clamp(
						shownLeft + dir * keyStep,
						leftRange.min,
						leftRange.max,
					);
					setLeft(next);
					saveLater(next, right);
				}}
			/>
			<main className="pane center">
				<DayPane />
			</main>
			{open && (
				<Splitter
					label="右ペインの幅"
					value={right}
					min={rightRange.min}
					max={rightRange.max}
					onStart={() => {
						dragStart.current = right;
						setDragging(true);
					}}
					// 右の境目は、左へ動かすと右ペインが広がる。範囲の端で止め、畳むのは閉じるボタンだけにする。
					onMove={(dx) =>
						setRight(clamp(dragStart.current - dx, rightRange.min, rightMax))
					}
					onEnd={() => {
						setDragging(false);
						save(left, right, collapsed);
					}}
					onKey={(dir) => {
						const next = clamp(right - dir * keyStep, rightRange.min, rightMax);
						setRight(next);
						saveLater(left, next);
					}}
				/>
			)}
			{/*
			 * 右の領域。開け閉めで幅をなめらかに変えるため、右ペインとバーを重ねて置き、見えている方だけを操作できるようにする。
			 * 境目をドラッグしているあいだは、幅が指に遅れないよう、なめらかに変えるのを止める。
			 */}
			<div
				className={dragging ? "right-dock dragging" : "right-dock"}
				style={{ width: open ? right : barWidth }}
			>
				<aside
					className={open ? "pane right" : "pane right hidden"}
					style={{ width: right }}
					inert={!open}
				>
					<RightPane out={out} onClose={closeRight} />
				</aside>
				<OutputBar
					ref={barRef}
					out={out}
					sheetOpen={sheetOpen}
					onToggle={sheetOpen ? closeRight : openRight}
					hidden={open}
				/>
			</div>
			{/*
			 * 狭いときは、右ペインをバーから左へ引き出して、中央の上に重ねる。バーは右端に残り、その最上段で開け閉めする。
			 * 暗い幕は掛けず、ヘッダーや左ペインはそのまま使える。
			 */}
			{auto && (
				<aside
					ref={sheetRef}
					className={sheetOpen ? "pane right sheet open" : "pane right sheet"}
					style={{ width: sheetWidth, right: barWidth }}
					inert={!sheetOpen}
				>
					<RightPane out={out} morphTarget={sheetOpen} />
				</aside>
			)}
			{out.dialogs}
		</div>
	);
}

// RightPane は右ペインの中身。並べて開いているときは、最上段に閉じるボタンだけを置く。
// 畳んだバーの開くボタンと同じ高さにそろえ、開け閉めの場所を変えないため。
// 狭いときに引き出したときは、バーの最上段で閉じるので、閉じるボタンを置かない。
function RightPane({
	out,
	onClose,
	morphTarget = false,
}: {
	out: Outputs;
	onClose?: () => void;
	// morphTarget は、バーのコピーと出力が移っていく先か (狭いときに引き出した右ペイン)。
	morphTarget?: boolean;
}) {
	return (
		<>
			{onClose && (
				<button
					type="button"
					className="pane-toggle"
					aria-label="右ペインを畳む"
					onClick={onClose}
				>
					<ChevronsRight />
				</button>
			)}
			<NotesSection />
			<OutputPane out={out} morphTarget={morphTarget} />
		</>
	);
}

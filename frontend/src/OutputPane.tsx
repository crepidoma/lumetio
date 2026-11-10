import { Button, Menu, Select, Text } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { Clipboard } from "@wailsio/runtime";
import { type ReactNode, type Ref, useEffect, useState } from "react";
import {
	type OutputChoices,
	type Preview,
	type PreviewOutput,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call, showInputError } from "./api";
import { ConfirmModal } from "./ConfirmModal";
import { useDay } from "./DayContext";
import { ChevronDown, ChevronsLeft, ChevronsRight } from "./icons";
import { morphNames } from "./morph";

// 確認のダイアログで待っている操作。
type Pending =
	| { kind: "warn"; then: () => void }
	| { kind: "exists"; files: string[] };

// Outputs は、右ペインと畳んだバーが共有する出力の状態と操作。
// 右ペインを畳んでも開いても、選んでいる出力と表示中のタブが変わらないよう、右ペインの外で持つ。
export type Outputs = {
	choices: OutputChoices | null;
	name: string | null;
	select: (name: string) => void;
	outputs: PreviewOutput[];
	current: PreviewOutput | undefined;
	setTab: (tab: number) => void;
	// previewError は、プレビュー全体を作れなかった理由 (出力セットが読めない出力を指しているなど)。
	previewError: string | null;
	copy: () => void;
	exportAll: () => void;
	// openFile は表示中の出力を出力先に書いたファイルを、openDir は出力先のフォルダを開く。
	openFile: () => void;
	openDir: () => void;
	// dialogs は確認のダイアログ。右ペインを畳んでいても出せるよう、ペインの外に置く。
	dialogs: ReactNode;
};

// hasChoice は、name が選べる出力セットか出力の中にあるか。
function hasChoice(c: OutputChoices, name: string): boolean {
	return !!(c.sets?.includes(name) || c.outputs?.includes(name));
}

// version は、出力と出力セット、既定の出力が変わるたびに増える。そのたびに一覧を読み直し、プレビューを作り直す。
export function useOutputs(
	onNotices: (messages: string[]) => void,
	version: number,
): Outputs {
	const { date, day } = useDay();
	const [choices, setChoices] = useState<OutputChoices | null>(null);
	const [name, setName] = useState<string | null>(null);
	const [preview, setPreview] = useState<Preview | null>(null);
	// 記録を書くたびに作り直すので、作れない理由は知らせではなくプレビューの場所に出し、知らせが積もらないようにする。
	const [previewError, setPreviewError] = useState<string | null>(null);
	const [tab, setTab] = useState(0);
	const [pending, setPending] = useState<Pending | null>(null);

	// biome-ignore lint/correctness/useExhaustiveDependencies: version は値を使わず、変わるたびに読み直すきっかけにする
	useEffect(() => {
		let stale = false;
		call(
			() => Service.OutputChoices(),
			() => {},
		).then((r) => {
			// 続けて読み直したとき、先に頼んだ古い一覧で新しい一覧を上書きしない。
			if (stale) return;
			// 読めなければ一覧をなくす。古いテンプレートのプレビューをコピーできるまま残さないため。
			if (!r.ok) {
				setChoices(null);
				return;
			}
			const c = r.value;
			setChoices(c);
			onNotices(c.notices ?? []);
			// 選んでいたものが消えたり名前が変わったりしていたら、既定に戻す。
			setName((n) => (n && hasChoice(c, n) ? n : c.default || null));
		});
		return () => {
			stale = true;
		};
	}, [onNotices, version]);

	// 選ぶものが変わったら、先頭のタブから見せる。
	// 前に選んでいたもののプレビューは消す。新しい結果が届くまで、別の出力の文面を見せたりコピーさせたりしないため。
	// biome-ignore lint/correctness/useExhaustiveDependencies: name は値を使わず、変わったときのきっかけにする
	useEffect(() => {
		setTab(0);
		setPreview(null);
		setPreviewError(null);
	}, [name]);

	// 記録を書くたびに day が変わるので、そのたびに作り直す。一覧を読み直したとき (テンプレートが変わったとき) も作り直す。
	useEffect(() => {
		// 選ぶものや一覧がなくなったら、前のプレビューを残さない。コピーで古い文面を出さないため。
		if (!name || !choices) {
			setPreview(null);
			setPreviewError(null);
			return;
		}
		// 一覧にない名前 (消えた、名前を変えた) では作らない。誤りが一瞬出ないよう、一覧から選び直すのを待つ。
		// 前のプレビューは消す。消した出力の文面をコピーできるまま残さないため。
		if (!hasChoice(choices, name)) {
			setPreview(null);
			setPreviewError(null);
			return;
		}
		if (day.date !== date) return;
		let stale = false;
		call(
			() => Service.Preview(date, name),
			() => {},
		).then((r) => {
			if (stale) return;
			if (r.ok) {
				setPreview(r.value);
				setPreviewError(null);
				onNotices(r.value.notices ?? []);
			} else {
				setPreview(null);
				setPreviewError(r.inputError ?? null);
			}
		});
		return () => {
			stale = true;
		};
	}, [date, day, name, choices, onNotices]);

	const outputs = preview?.outputs ?? [];
	const current = outputs[Math.min(tab, outputs.length - 1)];

	// 期間の外のプロジェクトか、計上する記録の時間未入力があれば、アプリの外へ出す前に確かめる。
	const confirmWarnings = (then: () => void) => {
		if ((day.warnings ?? []).length > 0) {
			setPending({ kind: "warn", then });
		} else {
			then();
		}
	};

	const copyNow = async () => {
		if (!current || current.error) return;
		try {
			await Clipboard.SetText(current.text);
		} catch (err) {
			notifications.show({
				color: "red",
				message: `コピーできませんでした: ${err instanceof Error ? err.message : String(err)}`,
				autoClose: false,
			});
			return;
		}
		notifications.show({
			message: `${current.name}をコピーしました`,
			autoClose: 2000,
		});
	};

	const exportNow = async (onConflict: "ask" | "overwrite" | "rename") => {
		if (!name) return;
		const r = await call(
			() => Service.Export(date, name, onConflict),
			() => {},
		);
		if (!r.ok) {
			showInputError(r.inputError);
			return;
		}
		const { written, exists, openErrors } = r.value;
		if ((exists ?? []).length > 0) {
			setPending({ kind: "exists", files: exists ?? [] });
			return;
		}
		notifications.show({
			message: `出力しました: ${(written ?? []).join("、")}`,
			autoClose: 4000,
		});
		for (const m of openErrors ?? []) {
			notifications.show({ color: "red", message: m, autoClose: false });
		}
	};

	// 出力したファイルとフォルダを開き直す。まだ書いていなければ、Go が開かずに知らせる。
	const openFile = async () => {
		if (!name || !current) return;
		const r = await call(
			() => Service.OpenOutputFile(date, name, current.name),
			() => {},
		);
		if (!r.ok) showInputError(r.inputError);
	};
	const openDir = async () => {
		const r = await call(
			() => Service.OpenOutputDir(),
			() => {},
		);
		if (!r.ok) showInputError(r.inputError);
	};

	const dialogs = (
		<>
			<ConfirmModal
				opened={pending?.kind === "warn"}
				// 警告は期間の外のプロジェクトと時間未入力の 2 種類あるので、見出しはどちらにも合う問いかけにし、中身は本文に並べる。
				title="続けますか？"
				onCancel={() => setPending(null)}
				actions={
					<Button
						onClick={() => {
							if (pending?.kind === "warn") pending.then();
							setPending(null);
						}}
					>
						続ける
					</Button>
				}
			>
				{(day.warnings ?? []).map((w) => (
					<Text key={w} size="sm">
						{w}
					</Text>
				))}
			</ConfirmModal>

			<ConfirmModal
				opened={pending?.kind === "exists"}
				title="出力先に同じ名前のファイルがあります"
				onCancel={() => setPending(null)}
				actions={
					<>
						<Button
							variant="default"
							onClick={() => {
								setPending(null);
								exportNow("rename");
							}}
						>
							別名で保存
						</Button>
						<Button
							onClick={() => {
								setPending(null);
								exportNow("overwrite");
							}}
						>
							上書き
						</Button>
					</>
				}
			>
				{pending?.kind === "exists" &&
					pending.files.map((f) => (
						<Text key={f} size="sm" ff="monospace">
							{f}
						</Text>
					))}
			</ConfirmModal>
		</>
	);

	return {
		choices,
		name,
		select: setName,
		outputs,
		current,
		setTab,
		previewError,
		copy: () => confirmWarnings(copyNow),
		exportAll: () => confirmWarnings(() => exportNow("ask")),
		openFile,
		openDir,
		dialogs,
	};
}

function canCopy(o: Outputs): boolean {
	return !!o.current && !o.current.error;
}

function canExport(o: Outputs): boolean {
	return !!o.name && o.outputs.length > 0;
}

// OutputPane は右ペインの出力。選んだ出力セットの出力をタブで並べ、プレビュー、コピー、出力を行う。
export function OutputPane({
	out,
	morphTarget,
}: {
	out: Outputs;
	morphTarget: boolean;
}) {
	if (!out.choices) return null;
	const data = [
		{ group: "出力セット", items: out.choices.sets ?? [] },
		{ group: "出力", items: out.choices.outputs ?? [] },
	].filter((g) => g.items.length > 0);
	const { current } = out;

	return (
		<section className="output">
			<Select
				size="sm"
				aria-label="出力セット"
				data={data}
				value={out.name}
				allowDeselect={false}
				onChange={(v) => v && out.select(v)}
			/>
			{out.outputs.length > 1 && (
				<div className="tabs" role="tablist">
					{out.outputs.map((o, i) => (
						<button
							key={o.name}
							type="button"
							role="tab"
							aria-selected={o === current}
							className={o === current ? "tab on" : "tab"}
							onClick={() => out.setTab(i)}
						>
							{o.name}
						</button>
					))}
				</div>
			)}
			{out.previewError && (
				<div className="preview error">{out.previewError}</div>
			)}
			{current &&
				(current.error ? (
					<div className="preview error">{current.error}</div>
				) : (
					<pre className="preview">{current.text}</pre>
				))}
			<div className="output-actions">
				<button
					type="button"
					className="text-button"
					disabled={!canCopy(out)}
					onClick={out.copy}
					style={morphTarget ? morphNames.copy : undefined}
				>
					コピー
				</button>
				<button
					type="button"
					className="primary-button"
					disabled={!canExport(out)}
					onClick={out.exportAll}
					style={morphTarget ? morphNames.export : undefined}
				>
					出力
				</button>
				{/* 出力したファイルとフォルダを開き直す。出力の続きの操作なので、出力の横に置く。 */}
				<Menu position="bottom-end">
					<Menu.Target>
						<button
							type="button"
							className="text-button output-more"
							aria-label="出力したものを開く"
						>
							<ChevronDown />
						</button>
					</Menu.Target>
					<Menu.Dropdown>
						<Menu.Item disabled={!canCopy(out)} onClick={out.openFile}>
							表示中のファイルを開く
						</Menu.Item>
						<Menu.Item onClick={out.openDir}>出力先のフォルダを開く</Menu.Item>
					</Menu.Dropdown>
				</Menu>
			</div>
		</section>
	);
}

// OutputBar は、右ペインを畳んだときの細いバー。最上段に開くボタン、下にコピーと出力のボタンと選んでいる出力の名前を出す。
// 開くボタンは、開いた右ペインの閉じるボタンと同じ高さにそろえる。
export function OutputBar({
	ref,
	out,
	sheetOpen,
	onToggle,
	hidden,
}: {
	// ref は、引き出した右ペインの外を押したかを見分けるために使う。
	ref?: Ref<HTMLElement>;
	out: Outputs;
	// sheetOpen は、狭いときに右ペインをバーから引き出しているか。引き出しているあいだは、バーで閉じる。
	sheetOpen: boolean;
	onToggle: () => void;
	// hidden は、右ペインを並べて開いていてバーを見せないとき。開け閉めの途中で重ねて薄くするため、消さずに隠す。
	hidden: boolean;
}) {
	// バーのコピーと出力が見えているときだけ、引き出す右ペインのコピーと出力へ移っていく元にする。
	const morphFrom = !sheetOpen && !hidden;
	return (
		<aside
			ref={ref}
			className={hidden ? "right-bar hidden" : "right-bar"}
			aria-label="出力 (畳んだ状態)"
			inert={hidden}
		>
			<button
				type="button"
				className="pane-toggle"
				aria-label={sheetOpen ? "右ペインを閉じる" : "右ペインを開く"}
				aria-expanded={sheetOpen}
				onClick={onToggle}
			>
				{sheetOpen ? <ChevronsRight /> : <ChevronsLeft />}
			</button>
			{/* コピーと出力は、開いた右ペインと同じく下に置く。引き出しているあいだは、右ペインに同じボタンがあるので隠す。 */}
			<div className="bar-bottom" hidden={sheetOpen}>
				<div className="bar-name">{out.name}</div>
				<button
					type="button"
					className="bar-button"
					disabled={!canCopy(out)}
					onClick={out.copy}
					style={morphFrom ? morphNames.copy : undefined}
				>
					コピー
				</button>
				<button
					type="button"
					className="bar-button primary"
					disabled={!canExport(out)}
					onClick={out.exportAll}
					style={morphFrom ? morphNames.export : undefined}
				>
					出力
				</button>
			</div>
		</aside>
	);
}

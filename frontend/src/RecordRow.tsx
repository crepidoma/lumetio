import {
	Button,
	Checkbox,
	Collapse,
	Group,
	Select,
	TagsInput,
	Textarea,
} from "@mantine/core";
import {
	type KeyboardEvent,
	type MouseEvent,
	type ReactNode,
	useState,
} from "react";
import {
	type Record,
	type RecordPatch,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { ConfirmModal } from "./ConfirmModal";
import { useDay } from "./DayContext";
import { FieldPopover } from "./FieldPopover";
import { noteMaxRows } from "./note";
import { projectLabel } from "./project";
import { TimeInput } from "./TimeInput";

// チップや時間の欄を押したときに、行の詳しい表示を開閉しないようにする。
const stop = (e: MouseEvent) => e.stopPropagation();

type Props = {
	record: Record;
	open: boolean;
	// onOpen は詳しい表示を開く記録の名前を渡す。null なら閉じる。
	onOpen: (title: string | null) => void;
	// handle は、並べ替えの取っ手。行の左端に置く。
	handle?: ReactNode;
};

export function RecordRow({ record, open, onOpen, handle }: Props) {
	const { date, act, focus } = useDay();

	const toggle = () => onOpen(open ? null : record.title);
	const onKeyDown = (e: KeyboardEvent) => {
		// 行から開いたダイアログ (portal で行の外に描く) のキーも React の木を通って届くので、受けない。
		// ダイアログの Esc で、ダイアログと一緒に詳しい表示まで閉じないようにするため。
		if (!e.currentTarget.contains(e.target as Node)) return;
		if (e.key === "Escape" && open) {
			onOpen(null);
			return;
		}
		// 行そのものにフォーカスがあるときだけ、Enter で開閉する。中の入力欄の Enter は奪わない。
		if (e.key === "Enter" && e.target === e.currentTarget) toggle();
	};

	return (
		// 行のどこを押しても開閉できるようにする。キーボードでは行にフォーカスして Enter で開閉する。
		// biome-ignore lint/a11y/noStaticElementInteractions: 中に入力欄やボタンがあるので button にできない。Enter で開閉できる。
		<div
			className={open ? "row record open" : "row record"}
			// biome-ignore lint/a11y/noNoninteractiveTabindex: 行を Tab で選び、Enter で開閉できるようにするため。
			tabIndex={0}
			onClick={toggle}
			onKeyDown={onKeyDown}
		>
			{handle}
			<div className="row-line">
				<TitleField
					title={record.title}
					onCommit={async (title) => {
						const message = await act(() =>
							Service.UpdateRecord(date, record.title, { title }),
						);
						// 名前を変えても、開いている詳しい表示を閉じない。Go は前後の空白を除いて NFC にそろえて書くので、同じ形で指す。
						if (!message && open) onOpen(title.trim().normalize("NFC"));
						return message;
					}}
				/>
				{(record.tags ?? []).map((t) => (
					<span key={t} className="badge">
						#{t}
					</span>
				))}
				{record.exclude && <span className="badge">計上しない</span>}
				{record.done && <span className="badge">完了</span>}
				<span className={record.total ? "row-total" : "row-total untimed"}>
					{record.total || "不定"}
				</span>
			</div>
			{record.project && (
				<div
					className={
						record.outOfPeriod ? "row-project out-of-period" : "row-project"
					}
				>
					{record.project.name}
				</div>
			)}
			{/* biome-ignore lint/a11y/noStaticElementInteractions: 中の操作で行が開閉しないよう、押した知らせを止めるだけ。 */}
			{/* biome-ignore lint/a11y/useKeyWithClickEvents: 同上。 */}
			<div className="row-times" onClick={stop}>
				{(record.times ?? []).map((t, i) => (
					<TimeChip
						// biome-ignore lint/suspicious/noArrayIndexKey: 時間は並びの位置で指すので、位置を key に入れる。
						key={`${i}-${t.label}`}
						title={record.title}
						index={i}
						label={t.label}
						input={t.input}
						value={t.value}
					/>
				))}
				<TimeInput
					focusSeq={focus?.title === record.title ? focus.seq : undefined}
					title={record.title}
					index={-1}
					onCommit={(input) =>
						act(() =>
							Service.AddRecords(date, [{ title: record.title, time: input }]),
						)
					}
				/>
			</div>
			{record.note && !open && <div className="row-note">{record.note}</div>}
			{/* 詳しい表示は、高さをなめらかに変えて開け閉めする。並べ替えで動かし始めたときに畳むのも、同じ動きにする。 */}
			<Collapse expanded={open} transitionDuration={200} keepMounted={false}>
				<RecordDetail record={record} />
			</Collapse>
		</div>
	);
}

// RecordDetail は、たまにしか直さない項目をまとめた詳しい表示。
function RecordDetail({ record }: { record: Record }) {
	const { date, run, projects, tags } = useDay();
	// 消すと時間やメモごと消え、元に戻せないので、確かめてから消す。
	const [removing, setRemoving] = useState(false);
	const update = (patch: RecordPatch) =>
		run(() => Service.UpdateRecord(date, record.title, patch));

	const data = projects.map((p) => ({ value: p.code, label: projectLabel(p) }));
	// 期間の外や一覧にないプロジェクトも、今の値として出す。
	if (
		record.project &&
		!projects.some((p) => p.code === record.project?.code)
	) {
		data.unshift({
			value: record.project.code,
			label: projectLabel(record.project),
		});
	}

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: 中の操作で行が閉じないよう、押した知らせを止めるだけ。
		// biome-ignore lint/a11y/useKeyWithClickEvents: 同上。
		<div className="row-detail" onClick={stop}>
			<Select
				label="プロジェクト"
				size="xs"
				data={data}
				value={record.project?.code ?? null}
				className={record.outOfPeriod ? "out-of-period" : undefined}
				searchable
				clearable
				nothingFoundMessage="ありません"
				onChange={(code) => update({ project: { code: code ?? "", name: "" } })}
			/>
			<TagsInput
				label="タグ"
				size="xs"
				data={tags}
				value={record.tags ?? []}
				onChange={(tags) => update({ tags })}
			/>
			<NoteField value={record.note} onCommit={(note) => update({ note })} />
			<Group justify="space-between">
				<Group gap="lg">
					<Checkbox
						size="xs"
						label="計上しない"
						checked={record.exclude}
						onChange={(e) => update({ exclude: e.currentTarget.checked })}
					/>
					{/* その日でそのタスクが終わったという印。付けると、直近のタスクに出なくなる。 */}
					<Checkbox
						size="xs"
						label="完了"
						checked={record.done}
						onChange={(e) => update({ done: e.currentTarget.checked })}
					/>
				</Group>
				<Button
					size="compact-xs"
					variant="subtle"
					color="red"
					onClick={() => setRemoving(true)}
				>
					記録を消す
				</Button>
			</Group>
			<ConfirmModal
				opened={removing}
				title={`${record.title} の記録を消しますか？`}
				onCancel={() => setRemoving(false)}
				actions={
					<Button
						color="red"
						onClick={() => {
							setRemoving(false);
							run(() => Service.RemoveRecord(date, record.title));
						}}
					>
						消す
					</Button>
				}
			/>
		</div>
	);
}

// TitleField は、ふだんは文字で出し、押すとその場で直せるタスク名。
function TitleField({
	title,
	onCommit,
}: {
	title: string;
	onCommit: (title: string) => Promise<string | null>;
}) {
	const [editing, setEditing] = useState(false);
	const [value, setValue] = useState(title);
	const [error, setError] = useState<string | null>(null);

	if (!editing) {
		return (
			<button
				type="button"
				className="row-title"
				onClick={(e) => {
					e.stopPropagation();
					setValue(title);
					setEditing(true);
				}}
			>
				{title}
			</button>
		);
	}
	const cancel = () => {
		setError(null);
		setEditing(false);
	};
	const finish = async () => {
		if (value === title) {
			cancel();
			return;
		}
		const message = await onCommit(value);
		setError(message);
		if (!message) setEditing(false);
	};
	return (
		<FieldPopover error={error}>
			<input
				className={error ? "row-title-input error" : "row-title-input"}
				// biome-ignore lint/a11y/noAutofocus: タスク名を押して直し始めたときだけ、その入力欄に移るため。
				autoFocus
				value={value}
				aria-label="タスク名"
				onClick={stop}
				onChange={(e) => setValue(e.currentTarget.value)}
				onBlur={finish}
				onKeyDown={(e) => {
					// 行の開閉に Enter や Esc を渡さない。
					e.stopPropagation();
					if (e.key === "Enter" && !e.nativeEvent.isComposing) {
						e.preventDefault();
						finish();
					} else if (e.key === "Escape") {
						cancel();
					}
				}}
			/>
		</FieldPopover>
	);
}

function NoteField({
	value,
	onCommit,
}: {
	value: string;
	onCommit: (note: string) => Promise<void>;
}) {
	const [draft, setDraft] = useState(value);
	const [shown, setShown] = useState(value);
	// 他で書き換わったら、編集中でない値を入れ替える。
	if (value !== shown) {
		setShown(value);
		setDraft(value);
	}
	return (
		<Textarea
			label="メモ"
			size="xs"
			autosize
			minRows={2}
			maxRows={noteMaxRows}
			value={draft}
			onChange={(e) => setDraft(e.currentTarget.value)}
			onBlur={() => {
				if (draft !== value) onCommit(draft);
			}}
		/>
	);
}

function TimeChip({
	title,
	index,
	label,
	input,
	value,
}: {
	title: string;
	index: number;
	label: string;
	// input は、直す欄に初めに入れる、時間の書き方として読める形。
	input: string;
	// value は保存形式。直すときに、どの時間を見ていたかを送る。
	value: string;
}) {
	const { date, act, run } = useDay();
	const [editing, setEditing] = useState(false);
	if (editing) {
		return (
			<TimeInput
				title={title}
				index={index}
				initial={input}
				autoFocus
				onCancel={() => setEditing(false)}
				onCommit={async (text) => {
					const message = await act(() =>
						Service.ReplaceTime(date, title, index, value, text),
					);
					if (!message) setEditing(false);
					return message;
				}}
			/>
		);
	}
	return (
		<span className="chip">
			<button
				type="button"
				className="chip-label"
				onClick={() => setEditing(true)}
			>
				{label}
			</button>
			<button
				type="button"
				className="chip-remove"
				aria-label="この時間を消す"
				onClick={() =>
					run(() => Service.ReplaceTime(date, title, index, value, ""))
				}
			>
				×
			</button>
		</span>
	);
}

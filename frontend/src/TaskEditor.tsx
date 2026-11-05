import {
	Autocomplete,
	Button,
	Checkbox,
	Chip,
	Group,
	Input,
	Modal,
	NumberInput,
	SegmentedControl,
	Select,
	Stack,
	TagsInput,
	TextInput,
} from "@mantine/core";
import { useEffect, useState } from "react";
import {
	type Project,
	Service,
	type TaskEdit,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call, showInputError, writeFailed } from "./api";
import { DateField } from "./DateField";
import { FieldPopover } from "./FieldPopover";
import { Star } from "./icons";
import { projectLabel } from "./project";
import {
	fromForm,
	type RepeatForm,
	type RepeatKind,
	repeatKinds,
	toForm,
	weekdays,
} from "./repeat";
import { usePreview } from "./usePreview";

type Props = {
	date: string;
	// projects は、その日に使える (期間の中の) プロジェクト。
	projects: Project[];
	// target は編集するタスクの名前。空文字なら新しく作る。null なら閉じている。
	target: string | null;
	onClose: () => void;
	// onChanged は、保存して書いた後に呼ぶ。予定や左ペインの一覧が変わるため。
	onChanged: () => void;
	reload: () => void;
};

// TaskEditor は、items.yaml のタスクの設定 (よく使う、いつもの時間、繰り返し) と、そのタスクを記録するときの値を直すダイアログ。
export function TaskEditor({
	date,
	projects,
	target,
	onClose,
	onChanged,
	reload,
}: Props) {
	const [loaded, setLoaded] = useState<TaskEdit | null>(null);

	useEffect(() => {
		setLoaded(null);
		if (target === null) return;
		if (target === "") {
			setLoaded(emptyTask);
			return;
		}
		let stale = false;
		call(() => Service.TaskEdit(target), reload).then((r) => {
			if (stale) return;
			if (r.ok) {
				setLoaded(r.value);
			} else {
				showInputError(r.inputError);
				onClose();
			}
		});
		return () => {
			stale = true;
		};
	}, [target, reload, onClose]);

	return (
		<Modal
			opened={target !== null}
			onClose={onClose}
			title={target === "" ? "新しいタスク" : "タスクの編集"}
			size="lg"
		>
			{loaded && target !== null && (
				<EditorForm
					key={target}
					date={date}
					projects={projects}
					original={target}
					initial={loaded}
					onClose={onClose}
					onChanged={onChanged}
					reload={reload}
				/>
			)}
		</Modal>
	);
}

const emptyTask: TaskEdit = {
	title: "",
	project: null,
	tags: [],
	exclude: false,
	time: "",
	repeat: null,
	favorite: false,
};

type FormProps = {
	date: string;
	projects: Project[];
	original: string;
	initial: TaskEdit;
	onClose: () => void;
	onChanged: () => void;
	reload: () => void;
};

function EditorForm({
	date,
	projects: projectsNow,
	original,
	initial,
	onClose,
	onChanged,
	reload,
}: FormProps) {
	const isNew = original === "";
	// プロジェクトの一覧は開いたときのものを使い続ける。他で一覧が変わって読み直されても、選んだ表示名から元のプロジェクトを引けるようにするため。
	// 開いたときにまだ一覧を読めていなければ、読めたものを使う。
	const [projects, setProjects] = useState(projectsNow);
	if (projects.length === 0 && projectsNow.length > 0) setProjects(projectsNow);
	const [title, setTitle] = useState(initial.title);
	const [projectText, setProjectText] = useState(
		initial.project ? projectLabel(initial.project) : "",
	);
	const [tags, setTags] = useState(initial.tags ?? []);
	const [exclude, setExclude] = useState(initial.exclude);
	const [time, setTime] = useState(initial.time);
	const [repeat, setRepeat] = useState<RepeatForm>(() =>
		toForm(initial.repeat),
	);
	const [favorite, setFavorite] = useState(initial.favorite);
	const [saving, setSaving] = useState(false);

	// 設定のないタスクは items.yaml に書かないので、新しく作るときは、設定を付けるまで保存できない。
	const hasSetting = favorite || time.trim() !== "" || repeat.kind !== "none";

	const projectOptions = projects.map(projectLabel);

	// 打った文字をプロジェクトにする。一覧の表示と同じなら一覧のもの、開いたときのままなら元の値、ほかはコードとして使う。
	const resolveProject = (): Project | null => {
		const text = projectText.trim();
		if (text === "") return null;
		const listed = projects.find((p) => projectLabel(p) === text);
		if (listed) return listed;
		if (initial.project && projectLabel(initial.project) === text)
			return initial.project;
		return { code: text, name: "" };
	};

	const setKind = (kind: RepeatKind) =>
		setRepeat((r) => ({
			...r,
			kind,
			// 隔週は数え始めが要るので、空なら開いている日から数える。
			start: kind === "biweekly" && r.start === "" ? date : r.start,
		}));

	const save = async () => {
		setSaving(true);
		const r = await call(
			() =>
				Service.SaveTask(original, isNew ? null : initial, {
					title,
					project: resolveProject(),
					tags,
					exclude,
					time,
					repeat: fromForm(repeat),
					favorite,
				}),
			reload,
		);
		setSaving(false);
		// 衝突したときは、開いたときの値が古くなったので、このまま保存し直しても衝突する。一覧を読み直して閉じ、読み直した値から開き直させる。
		if (r.ok || r.conflict) {
			onChanged();
			onClose();
		} else {
			if (writeFailed(r)) onChanged();
			showInputError(r.inputError);
		}
	};

	return (
		<Stack gap="md">
			<Group gap="xs" align="flex-end" wrap="nowrap">
				<TextInput
					label="タイトル"
					style={{ flex: 1 }}
					value={title}
					onChange={(e) => setTitle(e.currentTarget.value)}
					data-autofocus={isNew || undefined}
				/>
				{/* よく使うタスクへの出し入れは、左ペインと同じ☆で行う。 */}
				<button
					type="button"
					className={favorite ? "title-star on" : "title-star"}
					aria-label={
						favorite ? "よく使うタスクから外す" : "よく使うタスクに入れる"
					}
					aria-pressed={favorite}
					onClick={() => setFavorite((f) => !f)}
				>
					<Star size={18} filled={favorite} />
				</button>
			</Group>
			<Autocomplete
				label="プロジェクト"
				value={projectText}
				data={projectOptions}
				onChange={setProjectText}
			/>
			<TagsInput label="タグ" value={tags} onChange={setTags} />
			<Checkbox
				label="計上しない"
				checked={exclude}
				onChange={(e) => setExclude(e.currentTarget.checked)}
			/>
			<UsualTime value={time} onChange={setTime} />
			<Input.Wrapper label="繰り返し">
				<SegmentedControl
					fullWidth
					data={repeatKinds}
					value={repeat.kind}
					onChange={(v) => setKind(v as RepeatKind)}
				/>
			</Input.Wrapper>
			{(repeat.kind === "weekly" || repeat.kind === "biweekly") && (
				<WeekdayPicker
					multiple={repeat.kind === "weekly"}
					value={repeat.days}
					onChange={(days) => setRepeat((r) => ({ ...r, days }))}
				/>
			)}
			{repeat.kind === "monthly" && (
				<MonthlyPicker repeat={repeat} onChange={setRepeat} />
			)}
			{repeat.kind !== "none" && (
				<Group grow>
					<DateField
						label="いつから"
						value={repeat.start}
						onChange={(start) => setRepeat((r) => ({ ...r, start }))}
					/>
					<DateField
						label="いつまで"
						value={repeat.until}
						onChange={(until) => setRepeat((r) => ({ ...r, until }))}
					/>
				</Group>
			)}
			<Group justify="flex-end" gap="xs" mt="sm">
				<Button variant="default" onClick={onClose}>
					キャンセル
				</Button>
				<Button
					loading={saving}
					disabled={isNew && (!hasSetting || title.trim() === "")}
					onClick={save}
				>
					保存
				</Button>
			</Group>
		</Stack>
	);
}

// UsualTime は、いつもの時間の入力欄。記録の時間と同じく、打っているあいだに解釈した結果を出す。
function UsualTime({
	value,
	onChange,
}: {
	value: string;
	onChange: (v: string) => void;
}) {
	const [focused, setFocused] = useState(false);
	const [{ value: entries, error }] = usePreview(
		value,
		value.trim() === "" ? null : () => Service.PreviewUsualTime(value),
	);
	const preview = entries ? entries.map((e) => e.label).join("  ") : null;

	return (
		<FieldPopover error={error} hint={focused ? preview : null}>
			<TextInput
				label="いつもの時間"
				value={value}
				error={!!error}
				onFocus={() => setFocused(true)}
				onBlur={() => setFocused(false)}
				onChange={(e) => onChange(e.currentTarget.value)}
			/>
		</FieldPopover>
	);
}

function WeekdayPicker({
	multiple,
	value,
	onChange,
}: {
	multiple: boolean;
	value: string[];
	onChange: (days: string[]) => void;
}) {
	const chips = weekdays.map((w) => (
		<Chip key={w.code} value={w.code} size="sm">
			{w.label}
		</Chip>
	));
	return (
		<Group gap={6}>
			{multiple ? (
				<Chip.Group multiple value={value} onChange={onChange}>
					{chips}
				</Chip.Group>
			) : (
				<Chip.Group
					value={value[0] ?? null}
					onChange={(v) => onChange(v ? [v] : [])}
				>
					{chips}
				</Chip.Group>
			)}
		</Group>
	);
}

const nthOptions = [
	{ value: "1", label: "第1" },
	{ value: "2", label: "第2" },
	{ value: "3", label: "第3" },
	{ value: "4", label: "第4" },
	{ value: "5", label: "第5" },
	{ value: "-1", label: "最終" },
];

const weekdayOptions = weekdays.map((w) => ({
	value: w.code,
	label: `${w.label}曜`,
}));

function MonthlyPicker({
	repeat,
	onChange,
}: {
	repeat: RepeatForm;
	onChange: (f: (r: RepeatForm) => RepeatForm) => void;
}) {
	return (
		<Group gap="sm" align="center">
			<SegmentedControl
				data={[
					{ value: "nth", label: "曜日" },
					{ value: "day", label: "日付" },
				]}
				value={repeat.monthly}
				onChange={(v) =>
					onChange((r) => ({ ...r, monthly: v as RepeatForm["monthly"] }))
				}
			/>
			{repeat.monthly === "nth" ? (
				<>
					<Select
						w={90}
						data={nthOptions}
						value={repeat.nth}
						allowDeselect={false}
						onChange={(v) => v && onChange((r) => ({ ...r, nth: v }))}
						aria-label="週"
					/>
					<Select
						w={90}
						data={weekdayOptions}
						value={repeat.nthDay}
						allowDeselect={false}
						onChange={(v) => v && onChange((r) => ({ ...r, nthDay: v }))}
						aria-label="曜日"
					/>
				</>
			) : (
				<NumberInput
					w={90}
					min={1}
					max={31}
					clampBehavior="strict"
					allowDecimal={false}
					suffix="日"
					value={repeat.monthDay}
					onChange={(v) =>
						onChange((r) => ({
							...r,
							monthDay: typeof v === "number" ? v : "",
						}))
					}
					aria-label="日"
				/>
			)}
		</Group>
	);
}

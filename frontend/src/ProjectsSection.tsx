import {
	Autocomplete,
	Button,
	Checkbox,
	Group,
	Modal,
	SegmentedControl,
	Select,
	Stack,
	Text,
	TextInput,
} from "@mantine/core";
import { useEffect, useId, useState } from "react";
import {
	type ProjectRow,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call, showInputError } from "./api";
import { useStackedOverlay } from "./ConfirmModal";
import { DateField } from "./DateField";
import { AddRowButton, EditTable } from "./EditTable";
import { reorderShown } from "./project";
import {
	SaveButton,
	type SectionProps,
	useRows,
	useSave,
} from "./SettingsSection";

// ProjectsSection はプロジェクトの一覧。表で直し、CSV から取り込める。
// 期限切れ (読んだときに期限が今日より前だったもの) は、終わってもう使わないので、ふだんは隠し、出すときも末尾にまとめる。
// 直している途中で期限を変えても行が隠れないよう、期限切れかは読んだときの値で決める。
export function ProjectsSection({ onDirty, onSaved, reload }: SectionProps) {
	const t = useRows(Service.ProjectRows, reload);
	const { saving, error, save } = useSave(reload, t.rows);
	const [showExpired, setShowExpired] = useState(false);
	// csv は取り込むために読んだ CSV。null なら取り込みのダイアログを閉じている。
	const [csv, setCsv] = useState<string[][] | null>(null);
	useEffect(() => onDirty(t.dirty), [t.dirty, onDirty]);

	if (t.base === null) return null;
	const base = t.base;

	const chooseCSV = async () => {
		const path = await call(() => Service.ChooseCSV(), reload);
		if (!path.ok || !path.value) return;
		const file = path.value;
		const r = await call(() => Service.ReadCSV(file), reload);
		if (r.ok) {
			setCsv((r.value ?? []).map((row) => row ?? []));
		} else {
			showInputError(r.inputError);
		}
	};

	const expiredCount = t.items.filter((i) => i.row.expired).length;
	const shown = [
		...t.items.filter((i) => !i.row.expired),
		...(showExpired ? t.items.filter((i) => i.row.expired) : []),
	];

	return (
		<div className="table-section">
			<EditTable
				columns="project-columns"
				head={["コード", "名前", "開始日", "期限"]}
				items={shown}
				id={(i) => i.id}
				label={(i) => i.row.code || "コードのないプロジェクト"}
				onReorder={(order) => t.setItems((is) => reorderShown(is, order))}
				onRemove={(i) => t.remove(i.id)}
				foot={
					<Group justify="space-between">
						<Group gap="xs">
							<AddRowButton
								label="プロジェクトを足す"
								onClick={() =>
									t.add({
										code: "",
										name: "",
										start: "",
										until: "",
										expired: false,
									})
								}
							/>
							<Button variant="subtle" size="compact-sm" onClick={chooseCSV}>
								CSV から取り込む
							</Button>
						</Group>
						{expiredCount > 0 && (
							<Checkbox
								size="xs"
								label={`期限切れを表示する (${expiredCount})`}
								checked={showExpired}
								onChange={(e) => setShowExpired(e.currentTarget.checked)}
							/>
						)}
					</Group>
				}
			>
				{(i) => (
					<>
						<TextInput
							aria-label="コード"
							value={i.row.code}
							onChange={(e) => t.update(i.id, { code: e.currentTarget.value })}
						/>
						<TextInput
							aria-label="名前"
							value={i.row.name}
							onChange={(e) => t.update(i.id, { name: e.currentTarget.value })}
						/>
						<DateField
							aria-label="開始日"
							value={i.row.start}
							onChange={(start) => t.update(i.id, { start })}
						/>
						<DateField
							aria-label="期限"
							value={i.row.until}
							onChange={(until) => t.update(i.id, { until })}
						/>
					</>
				)}
			</EditTable>
			<SaveButton
				dirty={t.dirty}
				saving={saving}
				error={error}
				onSave={() =>
					save(
						() => Service.SaveProjects(base, t.rows),
						() => {
							onSaved();
							t.load();
						},
						t.load,
					)
				}
			/>
			<ImportDialog
				csv={csv}
				current={t.rows}
				onImported={(next) => {
					t.replace(next);
					setCsv(null);
				}}
				onClose={() => setCsv(null)}
				reload={reload}
			/>
		</div>
	);
}

// dateFormats は、日付の書式の候補。国によって月と日の並びが違うので、よく見る並びを出す。
const dateFormats = [
	"yyyy-MM-dd",
	"yyyy/MM/dd",
	"yyyy/M/d",
	"MM/dd/yyyy",
	"dd/MM/yyyy",
	"yyyyMMdd",
];

// unused は、名前や日付の列を使わないときの選択肢の値。
const unused = "-1";

type ImportProps = {
	current: ProjectRow[];
	onImported: (rows: ProjectRow[]) => void;
	onClose: () => void;
	reload: () => void;
};

// ImportDialog は、読んだ CSV からプロジェクトを取り込むダイアログ。csv が null なら閉じている。
// 設定のダイアログの上に開くので、同じ Modal.Stack に入って重なりの順と Esc を受ける順を決めてもらう。
// Modal.Stack は閉じたことを opened で知るので、閉じるときも Modal は外さずに opened を false にする。
// 外すと、閉じたはずのこのダイアログが一番上に残り、設定のダイアログが隠れたままになる。
function ImportDialog({
	csv,
	...props
}: ImportProps & { csv: string[][] | null }) {
	return (
		<Modal
			opened={csv !== null}
			onClose={props.onClose}
			title="CSV から取り込む"
			stackId={useId()}
			overlayProps={useStackedOverlay()}
		>
			{csv && <ImportForm csv={csv} {...props} />}
		</Modal>
	);
}

function ImportForm({
	csv,
	current,
	onImported,
	onClose,
	reload,
}: ImportProps & { csv: string[][] }) {
	const width = Math.max(0, ...csv.map((r) => r.length));
	const [skipHeader, setSkipHeader] = useState(true);
	const [code, setCode] = useState("0");
	const [name, setName] = useState(width > 1 ? "1" : unused);
	const [start, setStart] = useState(unused);
	const [until, setUntil] = useState(width > 2 ? "2" : unused);
	const [format, setFormat] = useState(dateFormats[0]);
	const [mode, setMode] = useState("add");
	const [error, setError] = useState<string | null>(null);

	// 列は、1 行目の値 (見出しか最初の値) を添えて見分けられるようにする。
	const columns = Array.from({ length: width }, (_, i) => ({
		value: String(i),
		label: csv[0]?.[i] ? `${i + 1} 列目: ${csv[0][i]}` : `${i + 1} 列目`,
	}));
	const optional = [{ value: unused, label: "使わない" }, ...columns];

	const run = async () => {
		const r = await call(
			() =>
				Service.ImportProjects(
					csv,
					{
						skipHeader,
						codeColumn: Number(code),
						nameColumn: Number(name),
						startColumn: Number(start),
						untilColumn: Number(until),
						dateFormat: format,
						replace: mode === "replace",
					},
					current,
				),
			reload,
		);
		if (r.ok) {
			onImported(r.value ?? []);
		} else if (r.inputError) {
			setError(r.inputError);
		}
	};

	return (
		<Stack gap="sm">
			<Checkbox
				label="1 行目は見出し"
				checked={skipHeader}
				onChange={(e) => setSkipHeader(e.currentTarget.checked)}
			/>
			<Select
				label="コードの列"
				data={columns}
				value={code}
				allowDeselect={false}
				onChange={(v) => v && setCode(v)}
			/>
			<Select
				label="名前の列"
				data={optional}
				value={name}
				allowDeselect={false}
				onChange={(v) => v && setName(v)}
			/>
			<Group gap="xs" wrap="nowrap" grow>
				<Select
					label="開始日の列"
					data={optional}
					value={start}
					allowDeselect={false}
					onChange={(v) => v && setStart(v)}
				/>
				<Select
					label="期限の列"
					data={optional}
					value={until}
					allowDeselect={false}
					onChange={(v) => v && setUntil(v)}
				/>
			</Group>
			<Autocomplete
				label="日付の書式"
				data={dateFormats}
				value={format}
				disabled={start === unused && until === unused}
				onChange={setFormat}
			/>
			<SegmentedControl
				value={mode}
				onChange={setMode}
				data={[
					{ value: "add", label: "今の一覧に足す" },
					{ value: "replace", label: "一覧ごと置き換える" },
				]}
			/>
			{error && (
				<Text c="red" size="sm">
					{error}
				</Text>
			)}
			<Group justify="flex-end" gap="xs">
				<Button variant="default" onClick={onClose}>
					キャンセル
				</Button>
				<Button onClick={run}>取り込む</Button>
			</Group>
		</Stack>
	);
}

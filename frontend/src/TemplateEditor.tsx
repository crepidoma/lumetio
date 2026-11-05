import {
	Checkbox,
	SegmentedControl,
	Select,
	Stack,
	Textarea,
	TextInput,
} from "@mantine/core";
import { type ReactNode, useEffect, useRef, useState } from "react";
import {
	type CSVColumn,
	type CSVField,
	type CSVSpec,
	Service,
	type TemplateDraft,
	type TextSpec,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { AddRowButton, EditTable } from "./EditTable";
import { sortedBy } from "./Sortable";
import { usePreview } from "./usePreview";

const encodings = [
	{ value: "utf-8", label: "UTF-8" },
	{ value: "utf-8-bom", label: "UTF-8 (BOM 付き)" },
];
const newlines = [
	{ value: "lf", label: "LF" },
	{ value: "crlf", label: "CRLF" },
];
const rowsOptions = [
	{ value: "projects", label: "プロジェクトごと" },
	{ value: "tasks", label: "記録ごと" },
];

type Props = {
	draft: TemplateDraft;
	onChange: (draft: TemplateDraft) => void;
	// outputs は、出力セットに入れられる出力の名前。
	outputs: string[];
	date: string;
};

// TemplateEditor は、出力か出力セット 1 つの中身を直す。
// 中身は「ファイル」「本文」などの小見出しで区切り、どの欄も左端をそろえる。
export function TemplateEditor({ draft, onChange, outputs, date }: Props) {
	return (
		<Stack gap="lg">
			<TextInput
				label="名前"
				value={draft.name}
				onChange={(e) => onChange({ ...draft, name: e.currentTarget.value })}
			/>
			{draft.kind === "text" && draft.text && (
				<TextFields
					spec={draft.text}
					onChange={(text) => onChange({ ...draft, text })}
				/>
			)}
			{draft.kind === "csv" && draft.csv && (
				<CSVFields
					spec={draft.csv}
					onChange={(csv) => onChange({ ...draft, csv })}
				/>
			)}
			{draft.kind === "set" && (
				<SetFields
					set={draft.set ?? []}
					outputs={outputs}
					onChange={(set) => onChange({ ...draft, set })}
				/>
			)}
			{draft.kind !== "set" && <DraftPreview draft={draft} date={date} />}
		</Stack>
	);
}

function Section({
	title,
	aside,
	children,
}: {
	title: string;
	aside?: ReactNode;
	children: ReactNode;
}) {
	return (
		<section className="editor-section">
			<div className="editor-section-head">
				<h3 className="editor-section-title">{title}</h3>
				{aside}
			</div>
			{children}
		</section>
	);
}

// FileFields は、テキストと CSV に共通する、書き出すファイルの設定。
// CSV の拡張子は csv に決まっているので、extension がなければ欄を出さない。
function FileFields({
	spec,
	onChange,
}: {
	spec: { file: string; extension?: string; encoding: string; newline: string };
	onChange: (patch: Partial<TextSpec & CSVSpec>) => void;
}) {
	const withExtension = spec.extension !== undefined;
	return (
		<Section title="ファイル">
			<div
				className={withExtension ? "file-fields with-extension" : "file-fields"}
			>
				<TextInput
					label="ファイル名の書式"
					classNames={{ input: "mono-input" }}
					value={spec.file}
					onChange={(e) => onChange({ file: e.currentTarget.value })}
				/>
				{withExtension && (
					<TextInput
						label="拡張子"
						value={spec.extension}
						onChange={(e) => onChange({ extension: e.currentTarget.value })}
					/>
				)}
				<Select
					label="文字コード"
					data={encodings}
					value={spec.encoding}
					allowDeselect={false}
					onChange={(v) => v && onChange({ encoding: v })}
				/>
				<Select
					label="改行"
					data={newlines}
					value={spec.newline}
					allowDeselect={false}
					onChange={(v) => v && onChange({ newline: v })}
				/>
			</div>
		</Section>
	);
}

function TextFields({
	spec,
	onChange,
}: {
	spec: TextSpec;
	onChange: (spec: TextSpec) => void;
}) {
	return (
		<>
			<FileFields
				spec={spec}
				onChange={(patch) => onChange({ ...spec, ...patch })}
			/>
			<Section title="本文">
				<Textarea
					aria-label="本文"
					classNames={{ input: "mono-input" }}
					autosize
					minRows={10}
					maxRows={20}
					value={spec.body}
					onChange={(e) => onChange({ ...spec, body: e.currentTarget.value })}
				/>
			</Section>
		</>
	);
}

// Column は CSV の列 1 つ。列の値は入力中に重なるので、並べ替えには id を使う。
type Column = { id: string; column: CSVColumn };

// custom は、一覧から選ばずにテンプレートをそのまま書く列の値。
const custom = "";

function CSVFields({
	spec,
	onChange,
}: {
	spec: CSVSpec;
	onChange: (spec: CSVSpec) => void;
}) {
	// 列の id は画面の中だけのもの。列の数が変わったときだけ振り直す。
	const ids = useRef<string[]>([]);
	const nextId = useRef(0);
	const columnsIn = spec.columns ?? [];
	while (ids.current.length < columnsIn.length) {
		ids.current.push(String(nextId.current++));
	}
	ids.current.length = columnsIn.length;
	const columns: Column[] = columnsIn.map((column, i) => ({
		id: ids.current[i],
		column,
	}));
	const setColumns = (next: Column[]) => {
		ids.current = next.map((c) => c.id);
		onChange({ ...spec, columns: next.map((c) => c.column) });
	};
	const update = (id: string, patch: Partial<CSVColumn>) =>
		setColumns(
			columns.map((c) =>
				c.id === id ? { ...c, column: { ...c.column, ...patch } } : c,
			),
		);

	// 選べる値は行の単位で変わるので、単位を変えるたびに Go から受け取る。
	const [fields, setFields] = useState<CSVField[]>([]);
	useEffect(() => {
		Service.CSVFields(spec.rows).then((fs) => setFields(fs ?? []));
	}, [spec.rows]);
	const fieldOptions = [
		...fields.map((f) => ({ value: f.key, label: f.label })),
		{ value: custom, label: "自由に書く" },
	];
	const chooseField = (c: Column, key: string) => {
		const f = fields.find((x) => x.key === key);
		update(c.id, { field: key, format: f?.defaultFormat ?? "" });
	};

	return (
		<>
			<FileFields
				spec={spec}
				onChange={(patch) => onChange({ ...spec, ...patch })}
			/>
			<Section title="行">
				<div className="csv-rows">
					<SegmentedControl
						data={rowsOptions}
						value={spec.rows}
						onChange={(rows) => onChange({ ...spec, rows })}
					/>
					<Checkbox
						label="見出しの行を出す"
						checked={spec.header}
						onChange={(e) =>
							onChange({ ...spec, header: e.currentTarget.checked })
						}
					/>
					<Checkbox
						label="計上する時間のない行を出さない"
						checked={spec.skipRowsWithoutCounted}
						onChange={(e) =>
							onChange({
								...spec,
								skipRowsWithoutCounted: e.currentTarget.checked,
							})
						}
					/>
				</div>
			</Section>
			<Section title="列">
				<EditTable
					columns="csv-columns"
					head={["見出し", "値"]}
					items={columns}
					id={(c) => c.id}
					label={(c, i) => c.column.header || `${i + 1} 列目`}
					onReorder={(order) =>
						setColumns(sortedBy(columns, order, (c) => c.id))
					}
					onRemove={(c) => setColumns(columns.filter((x) => x.id !== c.id))}
					foot={
						<AddRowButton
							label="列を足す"
							onClick={() =>
								setColumns([
									...columns,
									{
										id: String(nextId.current++),
										column: {
											header: "",
											field: "code",
											format: "",
											value: "",
										},
									},
								])
							}
						/>
					}
				>
					{(c) => (
						<>
							<TextInput
								aria-label="見出し"
								value={c.column.header}
								onChange={(e) =>
									update(c.id, { header: e.currentTarget.value })
								}
							/>
							<div className="csv-value">
								<Select
									className="csv-field"
									aria-label="値"
									data={fieldOptions}
									value={c.column.field}
									allowDeselect={false}
									onChange={(v) => v !== null && chooseField(c, v)}
								/>
								<ColumnDetail
									column={c.column}
									field={fields.find((f) => f.key === c.column.field)}
									onChange={(patch) => update(c.id, patch)}
								/>
							</div>
						</>
					)}
				</EditTable>
			</Section>
		</>
	);
}

// ColumnDetail は、値の横に出す欄。日付と時間なら書式、自由に書くならテンプレート。ほかは何も出さない。
function ColumnDetail({
	column,
	field,
	onChange,
}: {
	column: CSVColumn;
	field: CSVField | undefined;
	onChange: (patch: Partial<CSVColumn>) => void;
}) {
	if (column.field === custom) {
		return (
			<TextInput
				className="csv-detail"
				aria-label="テンプレート"
				placeholder="テンプレート"
				classNames={{ input: "mono-input" }}
				value={column.value}
				onChange={(e) => onChange({ value: e.currentTarget.value })}
			/>
		);
	}
	if (field?.format) {
		return (
			<TextInput
				className="csv-detail"
				aria-label="書式"
				placeholder="書式"
				classNames={{ input: "mono-input" }}
				value={column.format}
				onChange={(e) => onChange({ format: e.currentTarget.value })}
			/>
		);
	}
	return null;
}

function SetFields({
	set,
	outputs,
	onChange,
}: {
	set: string[];
	outputs: string[];
	onChange: (set: string[]) => void;
}) {
	const rest = outputs.filter((o) => !set.includes(o));
	return (
		<Section title="入れる出力">
			<EditTable
				columns="set-outputs"
				items={set}
				id={(name) => name}
				label={(name) => name}
				onReorder={onChange}
				onRemove={(name) => onChange(set.filter((n) => n !== name))}
				removeLabel="外す"
				foot={
					rest.length > 0 && (
						<Select
							className="set-add"
							size="xs"
							placeholder="出力を足す"
							aria-label="出力を足す"
							data={rest}
							value={null}
							onChange={(v) => v && onChange([...set, v])}
						/>
					)
				}
			>
				{(name) => <span>{name}</span>}
			</EditTable>
		</Section>
	);
}

// DraftPreview は、保存する前の中身で、開いている日の結果を出す。
function DraftPreview({ draft, date }: { draft: TemplateDraft; date: string }) {
	// 打つたびに作り直すと重いので、時間の解釈より長く待つ。
	const [{ value: preview, error: failed }] = usePreview(
		JSON.stringify([date, draft]),
		() => Service.PreviewTemplate(date, draft),
		300,
	);

	const error = failed ?? preview?.error;
	return (
		<Section
			title="プレビュー"
			aside={
				!error &&
				preview?.fileName && (
					<span className="editor-section-aside">{preview.fileName}</span>
				)
			}
		>
			<pre className={error ? "template-preview error" : "template-preview"}>
				{error || preview?.text || ""}
			</pre>
		</Section>
	);
}

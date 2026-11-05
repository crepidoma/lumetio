import { Button, Group, Menu, Select, Text } from "@mantine/core";
import { useCallback, useEffect, useRef, useState } from "react";
import {
	Service,
	type TemplateDraft,
	type TemplateEntry,
	type TemplateList,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call, showInputError } from "./api";
import { ConfirmModal } from "./ConfirmModal";
import { Plus } from "./icons";
import { TemplateEditor } from "./TemplateEditor";

type Props = {
	// date は開いている日。直している出力の結果を、この日の記録で作る。
	date: string;
	onDirty: (dirty: boolean) => void;
	onSaved: () => void;
	reload: () => void;
};

// Draft は直している出力か出力セット。base は読んだときのもので、新しく作るものは null。
type Draft = { base: TemplateEntry | null; value: TemplateDraft };

const kindLabels: Record<string, string> = {
	text: "テキスト",
	csv: "CSV",
	set: "出力セット",
};

function fromEntry(e: TemplateEntry): TemplateDraft {
	return { name: e.name, kind: e.kind, text: e.text, csv: e.csv, set: e.set };
}

// newDraft は、そのまま保存できる中身で始める新しい出力か出力セット。
function newDraft(
	kind: string,
	name: string,
	outputs: string[],
): TemplateDraft {
	switch (kind) {
		case "text":
			return {
				name,
				kind,
				text: {
					file: "yyyyMMdd",
					extension: "txt",
					encoding: "utf-8",
					newline: "lf",
					body: "",
				},
				csv: null,
				set: null,
			};
		case "csv":
			return {
				name,
				kind,
				text: null,
				csv: {
					file: "yyyyMMdd",
					encoding: "utf-8",
					newline: "crlf",
					header: true,
					rows: "projects",
					skipRowsWithoutCounted: false,
					columns: [{ header: "コード", field: "code", format: "", value: "" }],
				},
				set: null,
			};
		default:
			return { name, kind, text: null, csv: null, set: outputs.slice(0, 1) };
	}
}

// OutputsSection は出力と出力セットの管理。左に一覧、右に選んだものの中身を出す。
// 中身は 1 つずつ保存し、保存していない変更は別のものを選んでも残す。
export function OutputsSection({ date, onDirty, onSaved, reload }: Props) {
	const [list, setList] = useState<TemplateList | null>(null);
	// drafts は、直したものと新しく作ったもの。キーはファイル名か、新しいものなら new: で始まる名前。
	const [drafts, setDrafts] = useState<Record<string, Draft>>({});
	const [selected, setSelected] = useState<string | null>(null);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [deleting, setDeleting] = useState(false);
	const nextNew = useRef(0);

	const load = useCallback(
		(select?: string) => {
			call(() => Service.Templates(), reload).then((r) => {
				if (!r.ok) return;
				const entries = r.value.entries ?? [];
				setList(r.value);
				// ファイルがなくなった下書きは捨てる。選べず保存もできないのに、保存していない変更として残り続けるため。
				setDrafts((ds) =>
					Object.fromEntries(
						Object.entries(ds).filter(
							([k]) =>
								k.startsWith("new:") || entries.some((e) => e.fileName === k),
						),
					),
				);
				setSelected((s) => {
					const want = select ?? s;
					if (
						want &&
						(want.startsWith("new:") ||
							entries.some((e) => e.fileName === want))
					) {
						return want;
					}
					return entries[0]?.fileName ?? null;
				});
			});
		},
		[reload],
	);
	useEffect(() => load(), [load]);

	const entries = list?.entries ?? [];
	const isDirty = (key: string) => {
		const d = drafts[key];
		if (!d) return false;
		return (
			!d.base || JSON.stringify(d.value) !== JSON.stringify(fromEntry(d.base))
		);
	};
	const dirty = Object.keys(drafts).some(isDirty);
	useEffect(() => onDirty(dirty), [dirty, onDirty]);

	if (!list) return null;

	const readable = entries.filter((e) => !e.error);
	const sets = readable.filter((e) => e.kind === "set");
	const outputs = readable.filter((e) => e.kind !== "set").map((e) => e.name);
	const broken = entries.filter((e) => e.error);
	const newKeys = Object.keys(drafts).filter((k) => k.startsWith("new:"));

	const entry = entries.find((e) => e.fileName === selected) ?? null;
	const draft: Draft | null = selected
		? (drafts[selected] ??
			(entry && !entry.error ? { base: entry, value: fromEntry(entry) } : null))
		: null;

	const change = (value: TemplateDraft) => {
		if (!selected || !draft) return;
		setError(null);
		setDrafts((ds) => ({ ...ds, [selected]: { ...draft, value } }));
	};

	const add = (kind: string) => {
		const names = new Set([
			...entries.map((e) => e.name),
			...newKeys.map((k) => drafts[k].value.name),
		]);
		let name = `新しい${kindLabels[kind]}`;
		for (let i = 2; names.has(name); i++)
			name = `新しい${kindLabels[kind]} ${i}`;
		const key = `new:${nextNew.current++}`;
		setDrafts((ds) => ({
			...ds,
			[key]: { base: null, value: newDraft(kind, name, outputs) },
		}));
		setSelected(key);
		setError(null);
	};

	const dropDraft = (key: string) =>
		setDrafts((ds) => {
			const { [key]: _, ...rest } = ds;
			return rest;
		});

	const save = async () => {
		if (!selected || !draft) return;
		setSaving(true);
		const key = selected;
		const r = await call(
			() =>
				Service.SaveTemplate(
					draft.base?.fileName ?? "",
					draft.base?.data ?? "",
					draft.value,
				),
			reload,
		);
		setSaving(false);
		if (r.ok) {
			dropDraft(key);
			onSaved();
			load(r.value);
		} else if (r.conflict) {
			dropDraft(key);
			failed();
		} else if (r.inputError) {
			setError(r.inputError);
		} else {
			failed();
		}
	};

	// failed は、書けなかった後 (衝突と読み書きの失敗) に、一覧と右ペインを読み直す。途中まで書けていることがあり、
	// 自分で書いたファイルの監視の知らせは来ないので、ここで読み直さないと古い表示のまま残るため。
	const failed = () => {
		onSaved();
		load();
	};

	const remove = async () => {
		setDeleting(false);
		if (!selected) return;
		const key = selected;
		// まだ保存していないものは、下書きを捨てるだけ。
		if (key.startsWith("new:")) {
			dropDraft(key);
			setSelected(entries[0]?.fileName ?? null);
			return;
		}
		const target = entries.find((e) => e.fileName === key);
		if (!target) return;
		const r = await call(
			() => Service.DeleteTemplate(target.fileName, target.data),
			reload,
		);
		if (r.ok || r.conflict) {
			dropDraft(key);
			if (r.ok) onSaved();
			load();
		} else if (r.inputError) {
			showInputError(r.inputError);
		} else {
			failed();
		}
	};

	const saveDefault = async (name: string) => {
		const r = await call(
			() => Service.SaveDefaultOutput(list.default, name),
			reload,
		);
		if (r.ok) {
			onSaved();
			load();
		} else if (r.inputError) {
			showInputError(r.inputError);
		} else {
			// 衝突したときも、読んだときの既定が古いので、読み直さないと選び直しても衝突し続ける。
			failed();
		}
	};

	const item = (key: string, name: string, kind: string, warn = false) => (
		<button
			type="button"
			key={key}
			className={`template-item${key === selected ? " active" : ""}${warn ? " broken" : ""}`}
			onClick={() => {
				setSelected(key);
				setError(null);
			}}
		>
			<span className="template-item-name">
				{isDirty(key) ? "● " : ""}
				{name}
			</span>
			<span className="template-item-kind">{kindLabels[kind]}</span>
		</button>
	);

	const selectedName = draft?.value.name ?? entry?.name ?? "";

	return (
		<div className="outputs-section">
			<div className="outputs-list">
				<Select
					label="既定の出力"
					// 名前は出力と出力セットを通して重ならないので、名前をそのまま値にする。
					data={[
						{ value: "", label: "なし" },
						{ group: "出力セット", items: sets.map((s) => s.name) },
						{ group: "出力", items: outputs },
					]}
					value={
						readable.some((e) => e.name === list.default) ? list.default : ""
					}
					allowDeselect={false}
					onChange={(v) => v !== null && saveDefault(v)}
				/>
				<div className="outputs-items">
					{sets.map((e) => item(e.fileName, e.name, e.kind))}
					{readable
						.filter((e) => e.kind !== "set")
						.map((e) => item(e.fileName, e.name, e.kind))}
					{newKeys.map((k) =>
						item(k, drafts[k].value.name, drafts[k].value.kind),
					)}
					{broken.map((e) => item(e.fileName, e.fileName, e.kind, true))}
				</div>
				<Menu position="bottom-start">
					<Menu.Target>
						<Button variant="default" leftSection={<Plus />}>
							足す
						</Button>
					</Menu.Target>
					<Menu.Dropdown>
						<Menu.Item onClick={() => add("text")}>テキスト</Menu.Item>
						<Menu.Item onClick={() => add("csv")}>CSV</Menu.Item>
						<Menu.Item
							disabled={outputs.length === 0}
							onClick={() => add("set")}
						>
							出力セット
						</Menu.Item>
					</Menu.Dropdown>
				</Menu>
			</div>
			<div className="outputs-editor">
				{/* 中身が長くても保存と削除がいつも見えるよう、中身だけを送る。 */}
				<div className="outputs-editor-body">
					{entry?.error && (
						<Text c="red" size="sm">
							{entry.error}
						</Text>
					)}
					{draft && (
						<TemplateEditor
							// 選び直したら作り直す。前の出力のプレビューを、新しい結果が届くまで残さないため。
							key={selected}
							draft={draft.value}
							onChange={change}
							outputs={outputs}
							date={date}
						/>
					)}
				</div>
				{selected && (
					<Group
						justify="space-between"
						wrap="nowrap"
						className="outputs-editor-foot"
					>
						<Button
							variant="subtle"
							color="red"
							onClick={() => setDeleting(true)}
						>
							削除
						</Button>
						{draft && (
							<Group gap="md" wrap="nowrap">
								{error && (
									<Text c="red" size="sm">
										{error}
									</Text>
								)}
								<Button
									disabled={!isDirty(selected)}
									loading={saving}
									onClick={save}
								>
									保存
								</Button>
							</Group>
						)}
					</Group>
				)}
			</div>
			<ConfirmModal
				opened={deleting}
				title={`${selectedName} を削除しますか？`}
				onCancel={() => setDeleting(false)}
				actions={
					<Button color="red" onClick={remove}>
						削除する
					</Button>
				}
			/>
		</div>
	);
}

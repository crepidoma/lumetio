import {
	ActionIcon,
	Group,
	Modal,
	Stack,
	Tabs,
	TextInput,
} from "@mantine/core";
import { useCallback, useEffect, useState } from "react";
import {
	type Locations,
	type LocationsAtDefault,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call } from "./api";
import { DiscardConfirm } from "./ConfirmModal";
import { FieldPopover } from "./FieldPopover";
import { Folder, Undo } from "./icons";
import { NoteDefsSection } from "./NoteDefsSection";
import { OutputsSection } from "./OutputsSection";
import { ProjectsSection } from "./ProjectsSection";
import { SaveButton, type SectionProps, useSave } from "./SettingsSection";
import { TagsSection } from "./TagsSection";
import { usePreview } from "./usePreview";

type Props = {
	// date は開いている日。出力を直すときに、この日の記録で結果を出す。
	date: string;
	opened: boolean;
	onClose: () => void;
	// onLocationsChanged は、データフォルダや出力先を変えた後に呼ぶ。すべてを読み直すため。
	onLocationsChanged: () => void;
	// onOutputsChanged は、出力や出力セットを書いた後に呼ぶ。右ペインの一覧を読み直すため。
	onOutputsChanged: () => void;
	// reload は開いている日を読み直す。settings.yaml を書いた後は、日次画面の表示も変わるので呼ぶ。
	reload: () => void;
};

type Section =
	| "locations"
	| "projects"
	| "outputs"
	| "notes"
	| "tags"
	| "duration";

// SettingsDialog は設定画面。項目ごとに「保存」で書く。
export function SettingsDialog({
	date,
	opened,
	onClose,
	onLocationsChanged,
	onOutputsChanged,
	reload,
}: Props) {
	const [tab, setTab] = useState<Section>("locations");
	// dirty は、開いている項目に保存していない変更があるか。
	// ほかの項目へ移るときと閉じるときに確かめ、移ったら変更は破棄するので、開いている項目のぶんだけ持つ。
	const [dirty, setDirty] = useState(false);
	// leaving は、破棄してよいかを確かめているあいだの行き先。close なら閉じる。
	const [leaving, setLeaving] = useState<Section | "close" | null>(null);

	useEffect(() => {
		if (!opened) {
			setDirty(false);
			setTab("locations");
		}
	}, [opened]);

	// 見えていない項目は外すので、移ると前の項目の変更は消え、戻ったときは保存した値から始まる。
	const leave = (to: Section | "close") => {
		setLeaving(null);
		setDirty(false);
		if (to === "close") {
			onClose();
		} else {
			setTab(to);
		}
	};
	const requestLeave = (to: Section | "close") =>
		dirty ? setLeaving(to) : leave(to);

	return (
		// 確かめるダイアログと重ねるので、Esc を一番上のダイアログだけが受けるよう、同じ Modal.Stack に入れる。
		<Modal.Stack>
			<Modal
				stackId="settings"
				opened={opened}
				// Modal.Stack は一番上のダイアログの幕だけを出すが、上に重ねたときもこのダイアログの幕を残し、本体と見分けられるようにする。
				overlayProps={{ visible: opened }}
				onClose={() => requestLeave("close")}
				title="設定"
				size="min(1080px, calc(100vw - 64px))"
			>
				<Tabs
					className="settings"
					orientation="vertical"
					value={tab}
					onChange={(v) => v && v !== tab && requestLeave(v as Section)}
					// 見えていない項目は外す。戻ったときに、保存した値から読み直して始めるため。
					keepMounted={false}
				>
					<Tabs.List>
						<Tabs.Tab value="locations">場所</Tabs.Tab>
						<Tabs.Tab value="projects">プロジェクト</Tabs.Tab>
						<Tabs.Tab value="outputs">出力</Tabs.Tab>
						<Tabs.Tab value="notes">メモ欄</Tabs.Tab>
						<Tabs.Tab value="tags">タグ</Tabs.Tab>
						<Tabs.Tab value="duration">長さの表記</Tabs.Tab>
					</Tabs.List>
					<Tabs.Panel value="locations">
						<LocationsSection
							onDirty={setDirty}
							onSaved={onLocationsChanged}
							reload={reload}
						/>
					</Tabs.Panel>
					<Tabs.Panel value="projects">
						<ProjectsSection
							onDirty={setDirty}
							onSaved={reload}
							reload={reload}
						/>
					</Tabs.Panel>
					<Tabs.Panel value="outputs">
						<OutputsSection
							date={date}
							onDirty={setDirty}
							onSaved={onOutputsChanged}
							reload={reload}
						/>
					</Tabs.Panel>
					<Tabs.Panel value="notes">
						<NoteDefsSection
							onDirty={setDirty}
							onSaved={reload}
							reload={reload}
						/>
					</Tabs.Panel>
					<Tabs.Panel value="tags">
						<TagsSection onDirty={setDirty} onSaved={reload} reload={reload} />
					</Tabs.Panel>
					<Tabs.Panel value="duration">
						<DurationSection
							onDirty={setDirty}
							onSaved={reload}
							reload={reload}
						/>
					</Tabs.Panel>
				</Tabs>
			</Modal>
			<DiscardConfirm
				opened={leaving !== null}
				onCancel={() => setLeaving(null)}
				onDiscard={() => leaving && leave(leaving)}
			/>
		</Modal.Stack>
	);
}

function LocationsSection({ onDirty, onSaved, reload }: SectionProps) {
	const [base, setBase] = useState<Locations | null>(null);
	const [value, setValue] = useState<Locations | null>(null);
	const [defaults, setDefaults] = useState<Locations | null>(null);
	const { saving, error, save } = useSave(reload, value);

	useEffect(() => {
		call(() => Service.Locations(), reload).then((r) => {
			if (r.ok) {
				setBase(r.value);
				setValue(r.value);
			}
		});
		call(() => Service.DefaultLocations(), reload).then((r) => {
			if (r.ok) setDefaults(r.value);
		});
	}, [reload]);

	const dirty =
		!!base &&
		!!value &&
		(base.dataDir !== value.dataDir || base.outputDir !== value.outputDir);
	useEffect(() => onDirty(dirty), [dirty, onDirty]);

	// 書き方が違っても同じフォルダのことがあるので、初期値かどうかは Go でフォルダとして比べる。
	const [atDefault, setAtDefault] = useState<LocationsAtDefault | null>(null);
	useEffect(() => {
		if (!value) return;
		let stale = false;
		Service.AtDefault(value).then((r) => {
			if (!stale) setAtDefault(r);
		});
		return () => {
			stale = true;
		};
	}, [value]);

	if (!value) return null;

	const choose = async (key: keyof Locations, title: string) => {
		const r = await call(() => Service.ChooseFolder(title, value[key]), reload);
		if (r.ok && r.value) {
			const dir = r.value;
			setValue((v) => v && { ...v, [key]: dir });
		}
	};

	const reset = (key: keyof Locations) => {
		const dir = defaults?.[key];
		if (dir) setValue((v) => v && { ...v, [key]: dir });
	};
	// 初期値が分からないとき (空) と、もう初期値のフォルダを指しているときは戻せない。
	const canReset = (key: keyof Locations) =>
		!!defaults?.[key] && atDefault !== null && !atDefault[key];

	return (
		<Stack gap="md">
			<FolderField
				label="データフォルダ"
				description="記録と設定を置くフォルダ。"
				value={value.dataDir}
				onChoose={() => choose("dataDir", "データフォルダ")}
				canReset={canReset("dataDir")}
				onReset={() => reset("dataDir")}
			/>
			<FolderField
				label="出力先"
				description="出力したファイルを書き出すフォルダ。"
				value={value.outputDir}
				onChoose={() => choose("outputDir", "出力先")}
				canReset={canReset("outputDir")}
				onReset={() => reset("outputDir")}
			/>
			<SaveButton
				dirty={dirty}
				saving={saving}
				error={error}
				onSave={() =>
					save(
						() => Service.SaveLocations(value),
						() => {
							setBase(value);
							onSaved();
						},
						() => {},
					)
				}
			/>
		</Stack>
	);
}

function FolderField({
	label,
	description,
	value,
	onChoose,
	canReset,
	onReset,
}: {
	label: string;
	description: string;
	value: string;
	onChoose: () => void;
	canReset: boolean;
	onReset: () => void;
}) {
	return (
		<Group gap="xs" align="flex-end" wrap="nowrap">
			<TextInput
				label={label}
				description={description}
				value={value}
				readOnly
				style={{ flex: 1 }}
			/>
			<ActionIcon
				variant="default"
				size="input-sm"
				aria-label="選ぶ"
				onClick={onChoose}
			>
				<Folder />
			</ActionIcon>
			<ResetButton disabled={!canReset} onClick={onReset} />
		</Group>
	);
}

// ResetButton は、欄に初期値を入れる。書くのは「保存」。
function ResetButton({
	disabled,
	onClick,
}: {
	disabled: boolean;
	onClick: () => void;
}) {
	return (
		<ActionIcon
			variant="default"
			size="input-sm"
			aria-label="初期値に戻す"
			disabled={disabled}
			onClick={onClick}
		>
			<Undo />
		</ActionIcon>
	);
}

function DurationSection({ onDirty, onSaved, reload }: SectionProps) {
	const [base, setBase] = useState<string | null>(null);
	const [value, setValue] = useState("");
	const [initial, setInitial] = useState<string | null>(null);
	const { saving, error: saveError, save } = useSave(reload, value);
	const [focused, setFocused] = useState(false);

	const load = useCallback(() => {
		call(() => Service.DurationFormat(), reload).then((r) => {
			if (r.ok) {
				setBase(r.value);
				setValue(r.value);
			}
		});
	}, [reload]);
	useEffect(load, [load]);
	useEffect(() => {
		Service.DefaultDurationFormat().then(setInitial);
	}, []);

	const dirty = base !== null && value !== base;
	useEffect(() => onDirty(dirty), [dirty, onDirty]);

	const [{ value: samples, error }] = usePreview(value, () =>
		Service.PreviewDurationFormat(value),
	);

	if (base === null) return null;

	const hint = (samples ?? [])
		.map((s) => `${s.minutes}分 → ${s.text}`)
		.join("、");

	return (
		<Stack gap="md">
			<Group gap="xs" align="flex-end" wrap="nowrap">
				<FieldPopover error={error} hint={focused ? hint : null}>
					<TextInput
						label="長さの表記"
						description="画面に出す時間の長さの書き方。"
						value={value}
						error={!!error}
						style={{ flex: 1 }}
						onFocus={() => setFocused(true)}
						onBlur={() => setFocused(false)}
						onChange={(e) => setValue(e.currentTarget.value)}
					/>
				</FieldPopover>
				<ResetButton
					disabled={initial === null || value === initial}
					onClick={() => initial !== null && setValue(initial)}
				/>
			</Group>
			<SaveButton
				dirty={dirty}
				saving={saving}
				// 欄の吹き出しと同じ誤りは、ボタンの横に重ねて出さない。
				error={saveError === error ? null : saveError}
				onSave={() =>
					save(
						() => Service.SaveDurationFormat(base, value),
						() => {
							onSaved();
							load();
						},
						load,
					)
				}
			/>
		</Stack>
	);
}

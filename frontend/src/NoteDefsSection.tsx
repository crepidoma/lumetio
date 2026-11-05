import { Checkbox, Modal, Select, TextInput } from "@mantine/core";
import { useEffect, useState } from "react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { DiscardConfirm } from "./ConfirmModal";
import { AddRowButton, EditTable } from "./EditTable";
import {
	SaveButton,
	type SectionProps,
	useRows,
	useSave,
} from "./SettingsSection";

const types = [
	{ value: "text", label: "テキスト" },
	{ value: "bool", label: "真偽値" },
];

// NoteDefsSection はメモ欄の定義。設定画面と、日次画面のメモ欄の鉛筆のボタンから開く。
export function NoteDefsSection({ onDirty, onSaved, reload }: SectionProps) {
	const t = useRows(Service.NoteDefRows, reload);
	const { saving, error, save } = useSave(reload, t.rows);
	useEffect(() => onDirty(t.dirty), [t.dirty, onDirty]);

	if (t.base === null) return null;
	const base = t.base;

	return (
		<div className="table-section">
			<EditTable
				columns="notedefs-columns"
				head={["名前", "型", "前回の値を引き継ぐ"]}
				items={t.items}
				id={(i) => i.id}
				label={(i) => i.row.name || "名前のない欄"}
				onReorder={t.reorder}
				onRemove={(i) => t.remove(i.id)}
				foot={
					<AddRowButton
						label="欄を足す"
						onClick={() => t.add({ name: "", type: "text", carryOver: false })}
					/>
				}
			>
				{(i) => (
					<>
						<TextInput
							aria-label="名前"
							value={i.row.name}
							onChange={(e) => t.update(i.id, { name: e.currentTarget.value })}
						/>
						<Select
							aria-label="型"
							data={types}
							value={i.row.type}
							allowDeselect={false}
							onChange={(v) => v && t.update(i.id, { type: v })}
						/>
						<Checkbox
							aria-label="前回の値を引き継ぐ"
							checked={i.row.carryOver}
							onChange={(e) =>
								t.update(i.id, { carryOver: e.currentTarget.checked })
							}
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
						() => Service.SaveNoteDefs(base, t.rows),
						() => {
							onSaved();
							t.load();
						},
						t.load,
					)
				}
			/>
		</div>
	);
}

// NoteDefsDialog は、日次画面からメモ欄の定義を直すダイアログ。保存したら閉じる。
export function NoteDefsDialog({
	opened,
	onClose,
	reload,
}: {
	opened: boolean;
	onClose: () => void;
	reload: () => void;
}) {
	const [dirty, setDirty] = useState(false);
	const [confirming, setConfirming] = useState(false);
	const requestClose = () => (dirty ? setConfirming(true) : onClose());
	const close = () => {
		setDirty(false);
		onClose();
	};
	return (
		// 確かめるダイアログと重ねるので、Esc を一番上のダイアログだけが受けるよう、同じ Modal.Stack に入れる。
		<Modal.Stack>
			<Modal
				stackId="notedefs"
				opened={opened}
				onClose={requestClose}
				title="メモ欄"
				size="lg"
			>
				<NoteDefsSection
					onDirty={setDirty}
					onSaved={() => {
						reload();
						close();
					}}
					reload={reload}
				/>
			</Modal>
			<DiscardConfirm
				opened={confirming}
				onCancel={() => setConfirming(false)}
				onDiscard={() => {
					setConfirming(false);
					close();
				}}
			/>
		</Modal.Stack>
	);
}

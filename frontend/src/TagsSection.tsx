import { TextInput } from "@mantine/core";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { AddRowButton, EditTable } from "./EditTable";
import {
	SaveButton,
	type SectionProps,
	useReportDirty,
	useRows,
	useSave,
} from "./SettingsSection";

// TagsSection はタグの一覧。並びがタグの欄の候補の順になる。
export function TagsSection({ onDirty, onSaved, reload }: SectionProps) {
	const t = useRows(Service.TagRows, reload);
	const { saving, error, save } = useSave(reload, t.rows);
	useReportDirty(t.dirty, onDirty);

	if (t.base === null) return null;
	const base = t.base;

	return (
		<div className="table-section">
			<EditTable
				columns="tags-columns"
				head={["名前"]}
				items={t.items}
				id={(i) => i.id}
				label={(i) => i.row || "名前のないタグ"}
				onReorder={t.reorder}
				onRemove={(i) => t.remove(i.id)}
				foot={<AddRowButton label="タグを足す" onClick={() => t.add("")} />}
			>
				{(i) => (
					<TextInput
						aria-label="名前"
						value={i.row}
						onChange={(e) => t.set(i.id, e.currentTarget.value)}
					/>
				)}
			</EditTable>
			<SaveButton
				dirty={t.dirty}
				saving={saving}
				error={error}
				onSave={() =>
					save(
						() => Service.SaveTags(base, t.rows),
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

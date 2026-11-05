import { Button, Group, Text } from "@mantine/core";
import { useCallback, useEffect, useRef, useState } from "react";
import { call, writeFailed } from "./api";
import { sortedBy } from "./Sortable";

// SectionProps は、設定画面の各項目が受け取るもの。
export type SectionProps = {
	onDirty: (dirty: boolean) => void;
	onSaved: () => void;
	reload: () => void;
};

// useSave は「保存」を押したときの流れ。保存中か、保存できなかった入力の誤りを持つ。
// 保存できたら saved を、他で変わっていて保存できなかったら読み直す load を呼ぶ。
// 誤りは、入力 (input) を直したら消す。直した後も残っていると、まだ誤りがあるように見えるため。
export function useSave(reload: () => void, input: unknown) {
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const inputKey = JSON.stringify(input);
	// biome-ignore lint/correctness/useExhaustiveDependencies: inputKey は入力をまとめたもので、変わったら誤りを消すために入れている。
	useEffect(() => setError(null), [inputKey]);
	const save = async (
		op: () => Promise<unknown>,
		saved: () => void,
		load: () => void,
	) => {
		setSaving(true);
		const r = await call(op, reload);
		setSaving(false);
		setError(r.ok ? null : (r.inputError ?? null));
		if (r.ok) {
			saved();
		} else if (r.conflict) {
			load();
		} else if (writeFailed(r)) {
			reload();
		}
	};
	return { saving, error, save };
}

// SaveButton は項目の「保存」。保存できなかった入力の誤りは、押したボタンの横に出す。
export function SaveButton({
	dirty,
	saving,
	error,
	onSave,
}: {
	dirty: boolean;
	saving: boolean;
	error?: string | null;
	onSave: () => void;
}) {
	return (
		<Group justify="flex-end" mt="md" wrap="nowrap">
			{error && (
				<Text c="red" size="sm">
					{error}
				</Text>
			)}
			<Button disabled={!dirty} loading={saving} onClick={onSave}>
				保存
			</Button>
		</Group>
	);
}

// Row は設定の表の 1 行。名前などは入力中に空になったり重なったりするので、画面の中だけの id で見分ける。
export type Row<T> = { id: string; row: T };

// useRows は、Go から読んだ一覧を表で直すための状態。読んだときの一覧 (base) と比べて、変えたかを出す。
export function useRows<T>(
	fetch: () => Promise<T[] | null>,
	reload: () => void,
) {
	const [base, setBase] = useState<T[] | null>(null);
	const [items, setItems] = useState<Row<T>[]>([]);
	const nextId = useRef(0);
	const toItems = useCallback(
		(rows: T[]) => rows.map((row) => ({ id: String(nextId.current++), row })),
		[],
	);
	const load = useCallback(() => {
		call(fetch, reload).then((r) => {
			if (r.ok) {
				const rows = r.value ?? [];
				setBase(rows);
				setItems(toItems(rows));
			}
		});
	}, [fetch, reload, toItems]);
	useEffect(load, [load]);

	const rows = items.map((i) => i.row);
	const dirty = base !== null && JSON.stringify(base) !== JSON.stringify(rows);
	return {
		base,
		items,
		setItems,
		rows,
		dirty,
		load,
		// replace は、一覧ごと入れ替える (取り込んだときなど)。保存するまで base は変えない。
		replace: (next: T[]) => setItems(toItems(next)),
		add: (row: T) =>
			setItems((is) => [...is, { id: String(nextId.current++), row }]),
		update: (id: string, patch: Partial<T>) =>
			setItems((is) =>
				is.map((i) =>
					i.id === id ? { ...i, row: { ...i.row, ...patch } } : i,
				),
			),
		remove: (id: string) => setItems((is) => is.filter((i) => i.id !== id)),
		reorder: (order: string[]) =>
			setItems((is) => sortedBy(is, order, (i) => i.id)),
	};
}

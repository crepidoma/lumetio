import { ActionIcon, Button } from "@mantine/core";
import type { ReactNode } from "react";
import { Plus, Trash } from "./icons";
import { SortableItem, SortableList, sortedBy } from "./Sortable";

type Props<T> = {
	// columns は、見出しと行の列の幅を決めるクラス。取っ手と消すボタンの列も含める。
	columns: string;
	// head は見出し。取っ手と消すボタンの列のぶんは入れない。なければ見出しの行を出さない。
	head?: string[];
	items: T[];
	id: (item: T) => string;
	// label は、読み上げで行を指す名前。
	label: (item: T, index: number) => string;
	onReorder: (ids: string[]) => void;
	// children は、取っ手と消すボタンのあいだの欄を描く。
	children: (item: T, index: number) => ReactNode;
	onRemove: (item: T) => void;
	removeLabel?: string;
	// foot は、表の下端に置くもの (足すボタンなど)。
	foot?: ReactNode;
};

// EditTable は、設定で行を足し、消し、並べ替える表。
// 行が多いときは、見出しと下端を残して行だけを送る。
export function EditTable<T>({
	columns,
	head,
	items,
	id,
	label,
	onReorder,
	children,
	onRemove,
	removeLabel = "消す",
	foot,
}: Props<T>) {
	return (
		<div className="edit-table">
			{head && (
				<div className={`edit-table-head ${columns}`}>
					<span />
					{head.map((h) => (
						<span key={h}>{h}</span>
					))}
					<span />
				</div>
			)}
			<SortableList
				className="edit-table-body"
				ids={items.map(id)}
				onReorder={async (order) => onReorder(order)}
			>
				{(order) =>
					sortedBy(items, order, id).map((item, i) => (
						<SortableItem key={id(item)} id={id(item)} label={label(item, i)}>
							{(handle) => (
								<div className={`edit-table-row ${columns}`}>
									{handle}
									{children(item, i)}
									<ActionIcon
										variant="subtle"
										color="gray"
										aria-label={removeLabel}
										onClick={() => onRemove(item)}
									>
										<Trash />
									</ActionIcon>
								</div>
							)}
						</SortableItem>
					))
				}
			</SortableList>
			{foot && <div className="edit-table-foot">{foot}</div>}
		</div>
	);
}

// AddRowButton は、表の下端に置く、行を足すボタン。
export function AddRowButton({
	label,
	onClick,
}: {
	label: string;
	onClick: () => void;
}) {
	return (
		<Button
			variant="subtle"
			size="compact-sm"
			leftSection={<Plus />}
			onClick={onClick}
		>
			{label}
		</Button>
	);
}

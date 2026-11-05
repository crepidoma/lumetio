import {
	type CollisionDetection,
	closestCenter,
	DndContext,
	type DragEndEvent,
	KeyboardSensor,
	MeasuringStrategy,
	PointerSensor,
	pointerWithin,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	restrictToParentElement,
	restrictToVerticalAxis,
} from "@dnd-kit/modifiers";
import {
	arrayMove,
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { type ReactNode, useState } from "react";
import { Grip } from "./icons";

type ListProps = {
	ids: string[];
	// onReorder は、置いたときに新しい並びを渡し、保存し終えたら終わる。保存に失敗したら、呼び出し側が並びを読み直す。
	onReorder: (ids: string[]) => Promise<unknown>;
	// children は、今の並び (置いた直後は置いた並び) を受け取って行を描く。
	children: (order: string[]) => ReactNode;
	className?: string;
	// onDragStart は、つかんだ行の id を渡す (つかんだ行の詳しい表示を畳むためなど)。
	onDragStart?: (id: string) => void;
};

// 読み上げで使う説明。dnd-kit の初めの文は英語なので、日本語にする。
const accessibility = {
	screenReaderInstructions: {
		draggable:
			"Space でつかみ、上下キーで動かし、Space で置きます。Esc でやめます。",
	},
	announcements: {
		onDragStart: () => "つかみました。",
		onDragOver: () => "",
		onDragEnd: () => "置きました。",
		onDragCancel: () => "やめました。",
	},
};

// pointerFirst は、置き場所をマウスのいる位置の行で決める。
// 行の真ん中どうしで比べると、詳しい表示を開いて背の高くなった行は、一覧の上端まで動かしても真ん中が上の行を越えず、上へ置けないため。
// キーボードで動かすときはマウスの位置がないので、行の真ん中どうしで比べる。
const pointerFirst: CollisionDetection = (args) => {
	const hits = pointerWithin(args);
	return hits.length > 0 ? hits : closestCenter(args);
};

// SortableList は、縦に並んだ行をドラッグとキーボードで並べ替える。
// 縦にしか動かさず、一覧の外には出さない。
export function SortableList({
	ids,
	onReorder,
	children,
	className,
	onDragStart,
}: ListProps) {
	// 置いた後、保存の結果が届くまでのあいだも、置いた並びのまま見せる。元の並びに一度戻って見えないようにするため。
	const [pending, setPending] = useState<string[] | null>(null);
	const order = pending ?? ids;

	const sensors = useSensors(
		// 取っ手を押しただけ (クリック) ではつかまず、少し動かしたらつかむ。
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);

	const onDragEnd = ({ active, over }: DragEndEvent) => {
		if (!over || active.id === over.id) return;
		const next = arrayMove(
			order,
			order.indexOf(String(active.id)),
			order.indexOf(String(over.id)),
		);
		setPending(next);
		// 保存し終えたら (失敗しても)、届いた並びに戻す。
		onReorder(next).finally(() => setPending(null));
	};

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={pointerFirst}
			modifiers={[restrictToVerticalAxis, restrictToParentElement]}
			onDragStart={({ active }) => onDragStart?.(String(active.id))}
			onDragEnd={onDragEnd}
			// つかんだ後に行の高さが変わっても (詳しい表示を畳むなど)、置き場所がずれないよう、動かすあいだ測り直す。
			measuring={{ droppable: { strategy: MeasuringStrategy.Always } }}
			accessibility={accessibility}
		>
			<SortableContext items={order} strategy={verticalListSortingStrategy}>
				<div className={className}>{children(order)}</div>
			</SortableContext>
		</DndContext>
	);
}

// sortedBy は、items を ids の並びに並べる。置いた直後の並びで行を出すために使う。
export function sortedBy<T>(
	items: T[],
	ids: string[],
	id: (item: T) => string,
): T[] {
	return [...items].sort((a, b) => ids.indexOf(id(a)) - ids.indexOf(id(b)));
}

type ItemProps = {
	id: string;
	label: string;
	// children は、取っ手を受け取って行を描く。取っ手は行の左端に置く。
	children: (handle: ReactNode) => ReactNode;
};

// SortableItem は並べ替えられる行。取っ手だけでつかめる。
// 行のほかの場所 (タスク名、時間の欄など) の操作とぶつからないようにするため。
export function SortableItem({ id, label, children }: ItemProps) {
	const {
		attributes,
		listeners,
		setNodeRef,
		setActivatorNodeRef,
		transform,
		transition,
		isDragging,
	} = useSortable({ id });
	const handle = (
		<button
			type="button"
			ref={setActivatorNodeRef}
			className="drag-handle"
			aria-label={`${label}を並べ替える`}
			{...attributes}
			{...listeners}
			// 取っ手を押しても、行を押したことにしない (記録の行が開閉しないようにする)。
			onClick={(e) => e.stopPropagation()}
		>
			<Grip size={14} />
		</button>
	);
	return (
		<div
			ref={setNodeRef}
			className={isDragging ? "sortable dragging" : "sortable"}
			style={{ transform: CSS.Translate.toString(transform), transition }}
		>
			{children(handle)}
		</div>
	);
}

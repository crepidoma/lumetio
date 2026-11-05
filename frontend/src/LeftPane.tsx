import { type MouseEvent, type ReactNode, useState } from "react";
import type {
	Task,
	TaskLists,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { showInputError } from "./api";
import { useDay } from "./DayContext";
import { Check, List, Plus, Star } from "./icons";
import { SortableItem, SortableList, sortedBy } from "./Sortable";

type Props = {
	tasks: TaskLists;
	onFavorite: (title: string, favorite: boolean) => void;
	// onReorderFavorites は、よく使うタスクを新しい並びで保存する。
	onReorderFavorites: (titles: string[]) => Promise<void>;
	// onEdit はタスクの編集を開く。
	onEdit: (title: string) => void;
	// onComplete は、直近のタスクを完了にする。
	onComplete: (title: string) => void;
	// onManage はタスクの管理を開く。
	onManage: () => void;
};

export function LeftPane({
	tasks,
	onFavorite,
	onReorderFavorites,
	onEdit,
	onComplete,
	onManage,
}: Props) {
	const [othersOpen, setOthersOpen] = useState(false);
	const recent = tasks.recent ?? [];
	const favorites = tasks.favorites ?? [];
	const on = tasks.repeatingOn ?? [];
	const off = tasks.repeatingOff ?? [];
	const repeating = (t: Task) => (
		<TaskRow key={t.title} task={t} onFavorite={onFavorite} onEdit={onEdit} />
	);
	return (
		<>
			<nav className="tasks">
				{favorites.length > 0 && (
					<section className="tasks-section">
						<h2 className="tasks-head">よく使うタスク</h2>
						{/* よく使うタスクだけは並べ替えられる。並びが items.yaml の順になる。 */}
						<SortableList
							className="tasks-list"
							ids={favorites.map((t) => t.title)}
							onReorder={onReorderFavorites}
						>
							{(order) =>
								sortedBy(favorites, order, (t) => t.title).map((t) => (
									<SortableItem key={t.title} id={t.title} label={t.title}>
										{(handle) => (
											<TaskRow
												task={t}
												onFavorite={onFavorite}
												onEdit={onEdit}
												handle={handle}
											/>
										)}
									</SortableItem>
								))
							}
						</SortableList>
					</section>
				)}
				{recent.length > 0 && (
					<section className="tasks-section">
						<h2 className="tasks-head">直近のタスク</h2>
						{/* 直近のタスクは設定を持たないので、編集は開かない。設定を付けるときは☆でよく使うタスクにする。 */}
						{recent.map((t) => (
							<TaskRow
								key={t.title}
								task={t}
								onFavorite={onFavorite}
								onComplete={onComplete}
							/>
						))}
					</section>
				)}
				{on.length + off.length > 0 && (
					<section className="tasks-section">
						<h2 className="tasks-head">定期的なタスク</h2>
						{on.map(repeating)}
						{off.length > 0 && (
							<>
								{/* その日に予定の出ないものは、ふだんは畳んでおく。 */}
								<button
									type="button"
									className="tasks-fold"
									onClick={() => setOthersOpen((o) => !o)}
								>
									{othersOpen ? "▾" : "▸"} ほかの日 ({off.length})
								</button>
								{othersOpen && off.map(repeating)}
							</>
						)}
					</section>
				)}
			</nav>
			{/* タスクの管理は、タスクの一覧の近くで開けるよう、左ペインの最下段に置く。一覧をスクロールしても動かさない。 */}
			<div className="tasks-footer">
				<button type="button" className="tasks-manage" onClick={onManage}>
					<List size={14} />
					タスクの管理
				</button>
			</div>
		</>
	);
}

// 行の中のボタンを押したときに、行を押したこと (タスクの編集を開く) にしない。
const stop = (e: MouseEvent) => e.stopPropagation();

function TaskRow({
	task,
	onFavorite,
	onEdit,
	onComplete,
	handle,
}: {
	task: Task;
	onFavorite: Props["onFavorite"];
	// onEdit があれば、行を押すとタスクの編集を開く。items.yaml にあるタスクの行だけに渡す。
	onEdit?: Props["onEdit"];
	// onComplete があれば、完了のボタンを出す。直近のタスクの行だけに渡す。
	onComplete?: Props["onComplete"];
	// handle は、並べ替えの取っ手。よく使うタスクの行だけに付く。
	handle?: ReactNode;
}) {
	const { addTask } = useDay();
	const content = (
		<>
			{handle}
			<div className="task-text">
				<div className="task-title">{task.title}</div>
				{(task.project || task.recorded) && (
					<div className="task-meta">
						{task.recorded && <span className="task-recorded">記録あり</span>}
						{task.project && (
							<span className="task-project">{task.project.name}</span>
						)}
					</div>
				)}
			</div>
			{onComplete && (
				<button
					type="button"
					className="task-done"
					aria-label="完了"
					onClick={(e) => {
						stop(e);
						onComplete(task.title);
					}}
				>
					<Check size={14} />
				</button>
			)}
			<button
				type="button"
				className={task.favorite ? "task-star on" : "task-star"}
				aria-label={
					task.favorite ? "よく使うタスクから外す" : "よく使うタスクに入れる"
				}
				aria-pressed={task.favorite}
				onClick={(e) => {
					stop(e);
					onFavorite(task.title, !task.favorite);
				}}
			>
				<Star size={14} filled={task.favorite} />
			</button>
			<button
				type="button"
				className="task-add"
				aria-label="記録に足す"
				onClick={async (e) => {
					stop(e);
					showInputError(await addTask(task.title));
				}}
			>
				<Plus size={14} />
			</button>
		</>
	);
	if (!onEdit) return <div className="task">{content}</div>;
	return (
		// 行のどこを押しても、タスクの編集を開く。キーボードでは行にフォーカスして Enter で開く。
		// biome-ignore lint/a11y/noStaticElementInteractions: 中にボタンがあるので button にできない。Enter で開ける。
		<div
			className="task editable"
			// biome-ignore lint/a11y/noNoninteractiveTabindex: 行を Tab で選び、Enter で開けるようにするため。
			tabIndex={0}
			onClick={() => onEdit(task.title)}
			onKeyDown={(e) => {
				// 行そのものにフォーカスがあるときだけ開く。中のボタンの Enter は奪わない。
				if (e.key === "Enter" && e.target === e.currentTarget)
					onEdit(task.title);
			}}
		>
			{content}
		</div>
	);
}

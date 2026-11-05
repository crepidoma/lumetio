import { Button, Group, Modal } from "@mantine/core";
import { useEffect, useState } from "react";
import {
	type ManagedTask,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call } from "./api";
import { Star } from "./icons";
import { describe } from "./repeat";

type Props = {
	opened: boolean;
	onClose: () => void;
	// onEdit はタスクの編集を開く。空文字なら新しく作る。
	onEdit: (title: string) => void;
	// version は、タスクを書き換えるたびに変わる。一覧を読み直すため。
	version: number;
	reload: () => void;
};

// TaskManager は、items.yaml のすべてのタスクを並べる。
// まだ始まっていない定期的なタスクなど、左ペインに出ないタスクもここから直せる。
export function TaskManager({
	opened,
	onClose,
	onEdit,
	version,
	reload,
}: Props) {
	const [tasks, setTasks] = useState<ManagedTask[] | null>(null);

	// biome-ignore lint/correctness/useExhaustiveDependencies: version は中で使わないが、タスクを書き換えたときに一覧を読み直すために入れる。
	useEffect(() => {
		if (!opened) return;
		let stale = false;
		call(() => Service.ManagedTasks(), reload).then((r) => {
			if (!stale && r.ok) setTasks(r.value ?? []);
		});
		return () => {
			stale = true;
		};
	}, [opened, version, reload]);

	return (
		<Modal opened={opened} onClose={onClose} title="タスクの管理" size="xl">
			<Group justify="flex-end" mb="sm">
				<Button size="xs" onClick={() => onEdit("")}>
					新しいタスク
				</Button>
			</Group>
			<div className="managed">
				{(tasks ?? []).map((t) => (
					<button
						key={t.title}
						type="button"
						className="managed-row"
						onClick={() => onEdit(t.title)}
					>
						<span className={t.favorite ? "managed-star on" : "managed-star"}>
							{t.favorite && <Star size={14} filled />}
						</span>
						<span className="managed-title">
							{t.title}
							{t.project && (
								<span className="managed-project">{t.project.name}</span>
							)}
						</span>
						<span className="managed-time">
							{(t.time ?? []).map((e) => e.label).join("  ")}
						</span>
						<span className="managed-repeat">
							{t.repeat && describe(t.repeat)}
						</span>
					</button>
				))}
			</div>
		</Modal>
	);
}

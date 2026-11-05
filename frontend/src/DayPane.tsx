import { useState } from "react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useDay } from "./DayContext";
import { PlanRow } from "./PlanRow";
import { RecordRow } from "./RecordRow";
import { SortableItem, SortableList, sortedBy } from "./Sortable";

export function DayPane() {
	const { date, day, run } = useDay();
	// 詳しい表示は 1 行だけ開く。
	const [open, setOpen] = useState<string | null>(null);
	const records = day.records ?? [];
	const titles = records.map((r) => r.title);
	return (
		<div className="day">
			<div className="day-summary">
				<h1 className="day-title">記録</h1>
				<span className="day-total">
					合計 <b>{day.total}</b>
				</span>
			</div>
			{/* 記録は記録どうしで並べ替える。予定は items.yaml の順に出るもので、記録になっていないので動かさない。 */}
			<SortableList
				className="rows"
				ids={titles}
				onReorder={(next) => run(() => Service.ReorderRecords(date, next))}
				// つかんだ行の詳しい表示が開いていたら、動かし始めたときに畳む。背の高い行のままだと、動かしにくく、置き場所も見えにくいため。
				// ほかの行は畳まない。つかんだ行より上の行が縮むと、つかんだ行がカーソルから上へずれるため。
				onDragStart={(id) => setOpen((o) => (o === id ? null : o))}
			>
				{(order) =>
					sortedBy(records, order, (r) => r.title).map((r) => (
						<SortableItem key={r.title} id={r.title} label={r.title}>
							{(handle) => (
								<RecordRow
									record={r}
									open={open === r.title}
									onOpen={setOpen}
									handle={handle}
								/>
							)}
						</SortableItem>
					))
				}
			</SortableList>
			<div className="rows plans">
				{(day.plans ?? []).map((p) => (
					<PlanRow key={p.title} plan={p} />
				))}
			</div>
		</div>
	);
}

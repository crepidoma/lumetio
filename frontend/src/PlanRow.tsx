import {
	type Plan,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useDay } from "./DayContext";

export function PlanRow({ plan }: { plan: Plan }) {
	const { date, run } = useDay();
	return (
		<div className="row plan">
			<div className="row-line">
				<span className="row-title">{plan.title}</span>
				<span className="badge plan-badge">予定</span>
				{/* 押すことが多い実施済を目立たせ、スキップは控えめに後ろに置く。 */}
				<span className="plan-actions">
					<button
						type="button"
						className="primary-button"
						onClick={() =>
							run(() => Service.AddRecords(date, [{ title: plan.title }]))
						}
					>
						実施済
					</button>
					<button
						type="button"
						className="text-button"
						onClick={() => run(() => Service.SkipPlans(date, [plan.title]))}
					>
						スキップ
					</button>
				</span>
			</div>
			{plan.project && <div className="row-project">{plan.project.name}</div>}
			<div className="row-times">
				{(plan.times ?? []).map((t, i) => (
					<span
						// biome-ignore lint/suspicious/noArrayIndexKey: いつもの時間には同じ時間が並ぶこともあるので、位置を key に入れる。
						key={`${i}-${t.label}`}
						className="chip plan-chip"
					>
						{t.label}
					</span>
				))}
			</div>
		</div>
	);
}

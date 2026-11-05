import { Popover } from "@mantine/core";
import { DatePicker } from "@mantine/dates";
import { type ReactNode, useState } from "react";
import { addDays, calendarLabels, formatDay } from "./date";
import { ChevronLeft, ChevronRight, Gear } from "./icons";

type Props = {
	date: string;
	today: string;
	onChange: (date: string) => void;
	onSettings: () => void;
	// children は日付の右に置く検索欄。
	children?: ReactNode;
};

export function Header({ date, today, onChange, onSettings, children }: Props) {
	const [calendar, setCalendar] = useState(false);
	return (
		<header className="header">
			<div className="date-nav">
				<button
					type="button"
					className="icon-button"
					aria-label="前の日"
					onClick={() => onChange(addDays(date, -1))}
				>
					<ChevronLeft />
				</button>
				<Popover opened={calendar} onChange={setCalendar} position="bottom">
					<Popover.Target>
						<button
							type="button"
							className="date-label"
							onClick={() => setCalendar((o) => !o)}
						>
							{formatDay(date)}
						</button>
					</Popover.Target>
					<Popover.Dropdown>
						<DatePicker
							value={date}
							defaultDate={date}
							onChange={(d) => {
								if (d) {
									onChange(d);
									setCalendar(false);
								}
							}}
							{...calendarLabels}
						/>
					</Popover.Dropdown>
				</Popover>
				<button
					type="button"
					className="icon-button"
					aria-label="次の日"
					onClick={() => onChange(addDays(date, 1))}
				>
					<ChevronRight />
				</button>
				<button
					type="button"
					className="text-button"
					disabled={date === today}
					onClick={() => onChange(today)}
				>
					今日
				</button>
			</div>
			{children}
			<button
				type="button"
				className="icon-button settings-button"
				aria-label="設定"
				onClick={onSettings}
			>
				<Gear />
			</button>
		</header>
	);
}

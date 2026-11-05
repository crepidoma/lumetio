import { DatePickerInput, type DatePickerInputProps } from "@mantine/dates";
import { calendarLabels, formatDate } from "./date";

type Props = Omit<DatePickerInputProps, "value" | "onChange"> & {
	// value は yyyy-MM-dd。空なら日付を決めていない。
	value: string;
	onChange: (date: string) => void;
};

// DateField は、空にできる日付の欄。
export function DateField({ value, onChange, ...props }: Props) {
	return (
		<DatePickerInput
			{...props}
			value={value || null}
			clearable
			valueFormatter={({ date }) =>
				typeof date === "string" && date ? formatDate(date) : ""
			}
			onChange={(d) => onChange(d ?? "")}
			{...calendarLabels}
		/>
	);
}

import { useEffect, useRef, useState } from "react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useDay } from "./DayContext";
import { FieldPopover } from "./FieldPopover";
import { usePreview } from "./usePreview";

type Props = {
	title: string;
	// index が -1 なら記録に時間を足す欄、0 以上ならその位置の時間を直す欄。
	index: number;
	initial?: string;
	autoFocus?: boolean;
	// onCommit は Enter で呼ばれ、入力の誤りがあればその文を返す。
	onCommit: (input: string) => Promise<string | null>;
	onCancel?: () => void;
	// focusSeq が変わるたびに、この欄に移る。＋や検索で記録を足した後に、続けて時間を打てるようにするため。
	focusSeq?: number;
};

// TimeInput は時間の入力欄。入力中に、保存するときと同じ決まりで解釈した結果を出す。
export function TimeInput({
	title,
	index,
	initial = "",
	autoFocus,
	onCommit,
	onCancel,
	focusSeq,
}: Props) {
	const { date } = useDay();
	const ref = useRef<HTMLInputElement>(null);
	const [value, setValue] = useState(initial);

	useEffect(() => {
		if (focusSeq) ref.current?.focus();
	}, [focusSeq]);

	const [{ value: entries, error }, setPreview] = usePreview(
		JSON.stringify([date, title, index, value]),
		value.trim() === ""
			? null
			: () => Service.PreviewTime(date, title, index, value),
	);
	const preview = entries ? entries.map((e) => e.label).join("  ") : null;
	const clear = () => setPreview({ value: null, error: null });
	const variant = index < 0 ? "add" : "edit";

	const commit = async () => {
		// 直す欄を空にして確定したら、その時間を消す。足す欄は空なら何もしない。
		if (index < 0 && value.trim() === "") return;
		// 直す欄を何も変えずに確定したら、書かずに閉じる。
		if (index >= 0 && value === initial) {
			onCancel?.();
			return;
		}
		const message = await onCommit(value);
		if (message) {
			setPreview({ value: null, error: message });
			return;
		}
		setValue("");
	};

	return (
		<FieldPopover error={error} hint={preview}>
			<input
				ref={ref}
				className={
					error ? `time-input ${variant} error` : `time-input ${variant}`
				}
				placeholder={variant === "add" ? "+ 時間" : undefined}
				value={value}
				// biome-ignore lint/a11y/noAutofocus: チップを押して直し始めたときだけ、その入力欄に移るため。
				autoFocus={autoFocus}
				onChange={(e) => setValue(e.currentTarget.value)}
				onKeyDown={(e) => {
					if (e.key === "Enter" && !e.nativeEvent.isComposing) {
						e.preventDefault();
						commit();
					} else if (e.key === "Escape") {
						setValue(initial);
						clear();
						onCancel?.();
					}
				}}
				onBlur={() => {
					if (index >= 0) onCancel?.();
				}}
				aria-label="時間"
			/>
		</FieldPopover>
	);
}

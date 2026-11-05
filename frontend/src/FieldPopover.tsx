import { Popover, Text } from "@mantine/core";
import type { ReactElement } from "react";

type Props = {
	// error があれば赤い字で、なければ hint を出す。どちらもなければ閉じる。
	error?: string | null;
	hint?: string | null;
	children: ReactElement;
};

// FieldPopover は、入力欄の下に重ねて文を出す。
// 入力欄の下に行を足すと周りの行が動くので、重ねて出す。
export function FieldPopover({ error, hint, children }: Props) {
	const message = error || hint;
	return (
		<Popover
			opened={!!message}
			position="bottom-start"
			withArrow
			trapFocus={false}
			returnFocus={false}
			closeOnClickOutside={false}
			closeOnEscape={false}
		>
			<Popover.Target>{children}</Popover.Target>
			<Popover.Dropdown py={4} px={8}>
				<Text size="sm" c={error ? "red" : "dimmed"}>
					{message}
				</Text>
			</Popover.Dropdown>
		</Popover>
	);
}

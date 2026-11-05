import { Combobox, Text, TextInput, useCombobox } from "@mantine/core";
import { useHotkeys } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { useEffect, useRef, useState } from "react";
import {
	type SearchResult,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { Pencil, Search as SearchIcon } from "./icons";

type Props = {
	// onAdd はタスクをその日の記録に足し、入力の誤りがあればその文を返す。
	onAdd: (title: string) => Promise<string | null>;
	// onNotices は、読めずに除いたファイルの知らせを出す。
	onNotices: (messages: string[]) => void;
	// onEdit はタスクの編集を開く。
	onEdit: (title: string) => void;
};

// Ctrl+K (macOS では ⌘K) で検索欄に移る。
const shortcut = /Mac/.test(navigator.userAgent) ? "⌘K" : "Ctrl K";

// SearchBox は、過去のすべての記録と items.yaml からタスクを探し、その日の記録に足す欄。
// 一致するものがなければ、打った名前を新しいタスクとして足す。
export function SearchBox({ onAdd, onNotices, onEdit }: Props) {
	const combobox = useCombobox({
		onDropdownClose: () => combobox.resetSelectedOption(),
	});
	const input = useRef<HTMLInputElement>(null);
	const [query, setQuery] = useState("");
	// 一覧を作り終えたときに、そのあいだに打った分も含めて引き直せるよう、今の入力を持っておく。
	const latest = useRef(query);
	latest.current = query;
	const [results, setResults] = useState<SearchResult[]>([]);
	const [error, setError] = useState<string | null>(null);

	// 入力欄の中でも効くよう、無視するタグを空にする。
	// ダイアログを開いているあいだは、その下に隠れた検索欄へ移らない。
	useHotkeys(
		[
			[
				"mod+K",
				() => {
					if (document.querySelector('[role="dialog"][aria-modal="true"]'))
						return;
					input.current?.focus();
				},
			],
		],
		[],
	);

	useEffect(() => {
		let stale = false;
		Service.Search(query).then((r) => {
			if (!stale) setResults(r ?? []);
		});
		return () => {
			stale = true;
		};
	}, [query]);

	const start = () => {
		combobox.openDropdown();
		Service.StartSearch().then(
			(skipped) => {
				onNotices(skipped ?? []);
				// 一覧を作り直したので、今の入力で引き直す。
				const q = latest.current;
				Service.Search(q).then((r) => {
					if (latest.current === q) setResults(r ?? []);
				});
			},
			(err: Error) =>
				notifications.show({
					color: "red",
					message: err.message,
					autoClose: false,
				}),
		);
	};

	const add = async (title: string) => {
		if (title.trim() === "") return;
		const message = await onAdd(title);
		setError(message);
		if (message) return;
		setQuery("");
		combobox.closeDropdown();
	};

	const exact = results.some((r) => r.title === query.trim());
	return (
		<Combobox store={combobox} onOptionSubmit={(title) => add(title)}>
			<Combobox.Target>
				<TextInput
					ref={input}
					className="search"
					size="md"
					leftSection={<SearchIcon />}
					value={query}
					placeholder="タスクを探す・足す"
					aria-label="タスクを探す・足す"
					error={!!error}
					rightSection={<kbd className="kbd">{shortcut}</kbd>}
					rightSectionWidth={64}
					onFocus={start}
					onBlur={() => combobox.closeDropdown()}
					onChange={(e) => {
						setQuery(e.currentTarget.value);
						setError(null);
						combobox.openDropdown();
						combobox.resetSelectedOption();
					}}
					onKeyDown={(e) => {
						// 候補を選んでいなければ、打った名前をそのまま足す。
						if (
							e.key === "Enter" &&
							!e.nativeEvent.isComposing &&
							combobox.getSelectedOptionIndex() < 0
						) {
							e.preventDefault();
							add(query);
						} else if (e.key === "Escape") {
							setQuery("");
							setError(null);
							input.current?.blur();
						}
					}}
				/>
			</Combobox.Target>
			<Combobox.Dropdown>
				{/* 足せなかった理由は、入力欄のすぐ下に出ている候補の一覧の先頭に出す。周りの行を動かさないため。 */}
				{error && (
					<Combobox.Header>
						<Text size="sm" c="red">
							{error}
						</Text>
					</Combobox.Header>
				)}
				<Combobox.Options mah={360} style={{ overflowY: "auto" }}>
					{query.trim() !== "" && !exact && (
						<Combobox.Option value={query.trim()}>
							<Text size="sm">＋ 「{query.trim()}」を新しく足す</Text>
						</Combobox.Option>
					)}
					{results.map((r) => (
						<Combobox.Option key={r.title} value={r.title}>
							<div className="result">
								<span className="result-title">{r.title}</span>
								{r.project && (
									<span className="result-meta">{r.project.name}</span>
								)}
								{r.lastDate && (
									<span className="result-meta result-date">{r.lastDate}</span>
								)}
								{/* 行を選ぶと記録に足すので、タスクの編集は別のボタンで開く。編集するのは items.yaml にあるタスクだけ。 */}
								{r.inItems ? (
									<button
										type="button"
										className="result-edit"
										aria-label={`${r.title}を編集する`}
										// 入力欄からフォーカスを移さない。移ると候補の一覧が先に閉じ、押した知らせが届かないため。
										onMouseDown={(e) => e.preventDefault()}
										onClick={(e) => {
											// 行を選んだこと (記録に足すこと) にしない。
											e.stopPropagation();
											setQuery("");
											combobox.closeDropdown();
											input.current?.blur();
											onEdit(r.title);
										}}
									>
										<Pencil size={14} />
									</button>
								) : (
									// 鉛筆のない行も、日付の位置をそろえるため、同じ幅をあけておく。
									<span className="result-edit-space" />
								)}
							</div>
						</Combobox.Option>
					))}
				</Combobox.Options>
			</Combobox.Dropdown>
		</Combobox>
	);
}

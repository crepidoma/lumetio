import { ActionIcon, Switch, Textarea } from "@mantine/core";
import { useState } from "react";
import {
	type Note,
	Service,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useDay } from "./DayContext";
import { FieldPopover } from "./FieldPopover";
import { Pencil } from "./icons";
import { NoteDefsDialog } from "./NoteDefsSection";
import { noteMaxRows } from "./note";

// NotesSection はその日のメモ欄。型によらず、左に名前、右に値を並べる。
// 欄が 1 つもなくても、定義を直すボタンを出すために見出しは出す。
export function NotesSection() {
	const { date, day, reload } = useDay();
	const [editing, setEditing] = useState(false);
	const notes = day.notes ?? [];
	return (
		<section className="notes-section">
			<div className="notes-head">
				<h2 className="notes-title">メモ欄</h2>
				<ActionIcon
					variant="subtle"
					color="gray"
					size="sm"
					aria-label="メモ欄を直す"
					onClick={() => setEditing(true)}
				>
					<Pencil size={14} />
				</ActionIcon>
			</div>
			{notes.length > 0 && (
				<div className="notes">
					{notes.map((n) => (
						<div key={n.name} className="notes-row">
							<span className="notes-name">{n.name}</span>
							{n.type === "bool" ? (
								<BoolNote note={n} />
							) : (
								// 日を移ったら欄を作り直す。前の日の打ちかけの内容を、移った日に書かないため。
								<TextNote key={date} note={n} />
							)}
						</div>
					))}
				</div>
			)}
			<NoteDefsDialog
				opened={editing}
				onClose={() => setEditing(false)}
				reload={reload}
			/>
		</section>
	);
}

function BoolNote({ note }: { note: Note }) {
	const { date, run } = useDay();
	return (
		<Switch
			className="notes-switch"
			size="sm"
			checked={note.bool}
			aria-label={note.name}
			// 定義のない欄 (後で消した欄など) は、その日の値を見せるだけにする。
			disabled={!note.defined}
			onChange={(e) =>
				run(() => Service.SetBool(date, note.name, e.currentTarget.checked))
			}
		/>
	);
}

function TextNote({ note }: { note: Note }) {
	const { date, act } = useDay();
	const [draft, setDraft] = useState(note.text);
	const [shown, setShown] = useState(note.text);
	const [error, setError] = useState<string | null>(null);
	// 値が変わったら (書いた結果か、他で書き換わったもの)、欄を入れ替える。
	// 打ちかけの内容を残すと、欄を離れたときに他の PC の書き換えを上書きしてしまうため。
	if (note.text !== shown) {
		setShown(note.text);
		setDraft(note.text);
	}
	return (
		<FieldPopover error={error}>
			<Textarea
				className="notes-value"
				size="xs"
				autosize
				minRows={1}
				maxRows={noteMaxRows}
				value={draft}
				disabled={!note.defined}
				error={!!error}
				aria-label={note.name}
				onChange={(e) => {
					setDraft(e.currentTarget.value);
					// 打ち直したら、前の値の誤りは消す。残っていると、直した値にも誤りがあるように見えるため。
					setError(null);
				}}
				onKeyDown={(e) => {
					if (e.key === "Escape") setError(null);
				}}
				onBlur={async () => {
					if (draft === note.text) return;
					// 打ち始めたときの値も渡し、そのあいだに他で書き換わっていたら書かない。
					setError(
						await act(() => Service.SetText(date, note.name, shown, draft)),
					);
				}}
			/>
		</FieldPopover>
	);
}

import { notifications } from "@mantine/notifications";
import { Events } from "@wailsio/runtime";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { DataChanged } from "../bindings/events";
import {
	type Day,
	type Opened,
	type Project,
	Service,
	type TaskLists,
} from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { call, showInputError, writeFailed } from "./api";
import { DayContext, type DayContextValue } from "./DayContext";
import { Header } from "./Header";
import { LeftPane } from "./LeftPane";
import { SearchBox } from "./SearchBox";
import { SettingsDialog } from "./SettingsDialog";
import { TaskEditor } from "./TaskEditor";
import { TaskManager } from "./TaskManager";
import { useToday } from "./useToday";
import { Workspace } from "./Workspace";

function App() {
	const [date, setDate] = useState<string | null>(null);
	const [day, setDay] = useState<Day | null>(null);
	const [tasks, setTasks] = useState<TaskLists | null>(null);
	const [projects, setProjects] = useState<Project[]>([]);
	// focus は、時間を足す欄に移る記録。同じ記録に続けて移れるよう、回数も持つ。
	const [focus, setFocus] = useState<{ title: string; seq: number } | null>(
		null,
	);

	// 読めずに計算から除いたファイルは、計算とは別に知らせる。
	// 日を開くたびや検索欄に入るたびに同じ知らせが積もらないよう、同じ文は 1 回だけ出す。
	const shownNotices = useRef(new Set<string>());
	const notify = useCallback((messages: string[]) => {
		for (const m of messages) {
			if (shownNotices.current.has(m)) continue;
			shownNotices.current.add(m);
			notifications.show({ color: "orange", message: m, autoClose: false });
		}
	}, []);
	useEffect(() => notify(day?.notices ?? []), [day, notify]);
	useEffect(() => notify(tasks?.notices ?? []), [tasks, notify]);

	const today = useToday();
	// 最初に開くのは今日。その後に日付が変わっても、開いている日は移さない。
	if (date === null && today !== null) setDate(today);

	// current は今開いている日。日を移った後に前の日の結果が遅れて届いても、画面に出さないために使う。
	const current = useRef(date);
	current.current = date;

	// show は、開いた日の中身 (日、プロジェクト、左ペインの一覧) をまとめて画面に出す。
	// 左ペインの一覧は、その日の記録 (印や直近のタスク) で変わるので、日と一緒に受け取る。
	const show = useCallback((o: Opened) => {
		if (current.current !== o.day.date) return;
		setDay(o.day);
		setProjects(o.projects ?? []);
		setTasks(o.tasks);
	}, []);

	const reload = useCallback(() => {
		if (!date) return;
		call(() => Service.Open(date), reload).then((r) => {
			if (r.ok) show(r.value);
		});
	}, [date, show]);

	useEffect(reload, [reload]);

	const act = useCallback(
		async (op: () => Promise<Opened>) => {
			const r = await call(op, reload);
			if (r.ok) {
				show(r.value);
				return null;
			}
			if (writeFailed(r)) reload();
			return r.inputError ?? null;
		},
		[reload, show],
	);

	const run = useCallback(
		async (op: () => Promise<Opened>) => showInputError(await act(op)),
		[act],
	);

	const addTask = useCallback(
		async (title: string) => {
			if (!date) return null;
			const message = await act(() => Service.AddRecords(date, [{ title }]));
			if (!message) {
				// Go は名前を前後の空白を除いて NFC にそろえて書くので、同じ形で記録を指す。
				setFocus((f) => ({
					title: title.trim().normalize("NFC"),
					seq: (f?.seq ?? 0) + 1,
				}));
			}
			return message;
		},
		[date, act],
	);

	const setFavorite = useCallback(
		async (title: string, favorite: boolean) => {
			if (date) await run(() => Service.SetFavorite(date, title, favorite));
		},
		[date, run],
	);

	const reorderFavorites = useCallback(
		async (titles: string[]) => {
			if (date) await run(() => Service.ReorderFavorites(date, titles));
		},
		[date, run],
	);

	const completeTask = useCallback(
		async (title: string) => {
			if (!date) return;
			const r = await call(() => Service.CompleteTask(date, title), reload);
			if (r.ok || writeFailed(r)) {
				notify(r.ok ? (r.value ?? []) : []);
				// 開いている日の記録に印が付くこともあるので、その日ごと読み直す。
				reload();
			} else {
				showInputError(r.inputError);
			}
		},
		[date, reload, notify],
	);

	// editing は編集しているタスクの名前。空文字なら新しく作る。
	const [editing, setEditing] = useState<string | null>(null);
	const [managing, setManaging] = useState(false);
	// taskVersion は、タスクの設定を書き換えるたびに増やす。タスクの管理の一覧を読み直すため。
	const [taskVersion, setTaskVersion] = useState(0);
	const closeEditor = useCallback(() => setEditing(null), []);
	const taskChanged = useCallback(() => {
		// 予定や左ペインの一覧も変わるので、その日ごと読み直す。
		reload();
		setTaskVersion((v) => v + 1);
	}, [reload]);

	const [settingsOpen, setSettingsOpen] = useState(false);
	// dataVersion は、データフォルダや出力先を変えるたびに増やす。日次画面を作り直して、出力の一覧なども新しい場所から読むため。
	const [dataVersion, setDataVersion] = useState(0);
	const locationsChanged = useCallback(() => {
		setDataVersion((v) => v + 1);
		taskChanged();
	}, [taskChanged]);
	// outputsVersion は、出力と出力セットが変わるたびに増やし、右ペインに出力の一覧を読み直させる。
	// 日次画面ごと作り直すと、日次画面から開いたダイアログ (メモ欄の定義など) の入力中の内容が消えるため。
	const [outputsVersion, setOutputsVersion] = useState(0);
	const outputsChanged = useCallback(() => setOutputsVersion((v) => v + 1), []);

	// 他の PC や MCP がデータフォルダのファイルを書き換えたら、読み直す。知らせは出さない。
	// MCP が書いたときも同じ知らせが来るので、出すと AI に頼むたびに出てしまうため。
	// 開いているダイアログの中身は変えない。入力中の内容が勝手に変わらないようにするため。
	// 日次ファイルは、どの日が変わっても直近のタスクが変わりうるので、開いている日を読み直す。
	// 知らせの受け取りは一度だけ登録し、その時点の読み直しを reloadRef で呼ぶ。日を移るたびに登録し直さないため。
	const reloadRef = useRef(reload);
	reloadRef.current = reload;
	useEffect(
		() =>
			Events.On(DataChanged, (ev) => {
				const c = ev.data;
				// 既定の出力は settings.yaml にあるので、settings が変わったときも出力の一覧を読み直す。
				if (c.templates || c.settings) outputsChanged();
				if (c.settings || c.items || (c.dates ?? []).length > 0) {
					reloadRef.current();
				}
			}),
		[outputsChanged],
	);

	const value = useMemo<DayContextValue | null>(
		// 日を移った直後は、前の日の中身を新しい日として出さない。
		() =>
			date && day && day.date === date
				? { date, day, projects, act, run, addTask, focus, reload }
				: null,
		[date, day, projects, act, run, addTask, focus, reload],
	);
	// 新しい日を読み込むあいだは、前の日の中身 (前の日の日付と操作の組) をそのまま出しておく。
	// ペインをいったん外すと、引き出した右ペインや出力の選択などの状態が消えてしまうため。
	const lastValue = useRef<DayContextValue | null>(null);
	if (value) lastValue.current = value;
	const shownValue = value ?? lastValue.current;

	if (!today || !date) {
		return null;
	}
	return (
		<div className="layout">
			<Header
				date={date}
				today={today}
				onChange={(d) => {
					// 日を移ったら、前の日に足した記録の欄へは移らない。
					setFocus(null);
					setDate(d);
				}}
				onSettings={() => setSettingsOpen(true)}
			>
				<SearchBox onAdd={addTask} onNotices={notify} onEdit={setEditing} />
			</Header>
			{shownValue && (
				<DayContext.Provider value={shownValue}>
					<Workspace
						key={dataVersion}
						outputsVersion={outputsVersion}
						left={
							tasks && (
								<LeftPane
									tasks={tasks}
									onFavorite={setFavorite}
									onReorderFavorites={reorderFavorites}
									onEdit={setEditing}
									onComplete={completeTask}
									onManage={() => setManaging(true)}
								/>
							)
						}
						onNotices={notify}
					/>
				</DayContext.Provider>
			)}
			<TaskManager
				opened={managing}
				onClose={() => setManaging(false)}
				onEdit={setEditing}
				version={taskVersion}
				reload={reload}
			/>
			<SettingsDialog
				date={date}
				opened={settingsOpen}
				onClose={() => setSettingsOpen(false)}
				onLocationsChanged={locationsChanged}
				onOutputsChanged={outputsChanged}
				reload={reload}
			/>
			<TaskEditor
				date={date}
				projects={projects}
				target={editing}
				onClose={closeEditor}
				onChanged={taskChanged}
				reload={reload}
			/>
		</div>
	);
}

export default App;

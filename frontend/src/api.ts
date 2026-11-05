import { notifications } from "@mantine/notifications";

// Go がエラーと一緒に渡す種類。
type ErrorKind = "input" | "conflict" | "internal";

function kindOf(err: unknown): ErrorKind {
	const cause = (err as { cause?: { kind?: ErrorKind } } | null)?.cause;
	return cause?.kind ?? "internal";
}

function messageOf(err: unknown): string {
	return err instanceof Error ? err.message : String(err);
}

// Result は操作の結果。入力の誤りは呼び出し側に返して入力欄に出させ、衝突したかも返す (読み直した後に、古い値のまま続けないため)。
export type Result<T> =
	| { ok: true; value: T }
	| { ok: false; inputError?: string; conflict?: boolean };

// showInputError は、入力の誤りを出す欄のない操作で、誤りを知らせで出す。
export function showInputError(message: string | null | undefined) {
	if (message) notifications.show({ color: "red", message });
}

// writeFailed は、書く操作が、入力の誤りでも衝突でもなく失敗したか。
// そのときは途中まで書けていることがあり、自分で書いたファイルの監視の知らせは来ないので、呼び出し側で読み直す。
export function writeFailed(r: Result<unknown>): boolean {
	return !r.ok && !r.inputError && !r.conflict;
}

// call は Go の操作を呼ぶ。
// 衝突は読み直して短く知らせ、読み書きの失敗は消すまで残る知らせで出す。
export async function call<T>(
	op: () => Promise<T>,
	reload: () => void,
): Promise<Result<T>> {
	try {
		return { ok: true, value: await op() };
	} catch (err) {
		switch (kindOf(err)) {
			case "input":
				return { ok: false, inputError: messageOf(err) };
			case "conflict":
				notifications.show({
					message: "他で変更されたため、読み直しました",
					autoClose: 4000,
				});
				reload();
				return { ok: false, conflict: true };
			default:
				notifications.show({
					color: "red",
					message: messageOf(err),
					autoClose: false,
				});
				return { ok: false };
		}
	}
}

import { useCallback, useEffect, useRef, useState } from "react";

export type Preview<T> = { value: T | null; error: string | null };

const none = { value: null, error: null };

// usePreview は、入力中の値で Go に結果 (時間の解釈やテンプレートの結果) を作らせる。
// 打つたびに Go を呼ばないよう、少し待ってから呼ぶ。入力が変わった後に届いた前の結果は捨てる。
// key は入力をまとめた文字列で、変わるたびに作り直す。run が null なら呼ばずに空にする (欄が空のときなど)。
export function usePreview<T>(
	key: string,
	run: (() => Promise<T>) | null,
	delay = 150,
): [Preview<T>, (p: Preview<T>) => void] {
	const [preview, setShown] = useState<Preview<T>>(none);
	// generation は、出すものを決めた回数。呼び出し側が誤りなどを出した後に、待っていた前の結果で上書きしないため。
	const generation = useRef(0);
	const setPreview = useCallback((p: Preview<T>) => {
		generation.current++;
		setShown(p);
	}, []);
	// run は描くたびに作り直されるので、呼ぶときの最新のものを使い、作り直すきっかけは key だけにする。
	const runRef = useRef(run);
	runRef.current = run;
	const skip = run === null;

	// biome-ignore lint/correctness/useExhaustiveDependencies: key は入力をまとめたもので、変わったら作り直すために入れている。
	useEffect(() => {
		const op = runRef.current;
		if (skip || !op) {
			setShown(none);
			return;
		}
		const mine = ++generation.current;
		const timer = setTimeout(() => {
			op().then(
				(value) => {
					if (mine === generation.current) setShown({ value, error: null });
				},
				(err: Error) => {
					if (mine === generation.current)
						setShown({ value: null, error: err.message });
				},
			);
		}, delay);
		return () => {
			generation.current++;
			clearTimeout(timer);
		};
	}, [key, skip, delay]);

	return [preview, setPreview];
}

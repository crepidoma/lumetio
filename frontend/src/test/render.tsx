import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import type { ReactNode } from "react";
import { render } from "vitest-browser-react";
import { theme } from "../theme";

// renderApp は、アプリと同じテーマと知らせの置き場の中に部品を描く。
export function renderApp(ui: ReactNode) {
	return render(ui, { wrapper: Providers });
}

export function Providers({ children }: { children: ReactNode }) {
	return (
		<MantineProvider theme={theme}>
			<Notifications />
			{children}
		</MantineProvider>
	);
}

// inputError は、Go が入力の誤りとして返すエラーと同じ形のもの。
export function inputError(message: string): Error {
	return new Error(message, { cause: { kind: "input" } });
}

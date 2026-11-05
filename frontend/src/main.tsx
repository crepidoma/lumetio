import "@mantine/core/styles.css";
import "@mantine/dates/styles.css";
import "@mantine/notifications/styles.css";
import "./app.css";

import { MantineProvider } from "@mantine/core";
import { DatesProvider } from "@mantine/dates";
import { Notifications } from "@mantine/notifications";
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { theme } from "./theme";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<React.StrictMode>
		<MantineProvider theme={theme} defaultColorScheme="auto">
			{/* 週は月曜から始める (繰り返しの週の数え方と同じ)。 */}
			<DatesProvider settings={{ firstDayOfWeek: 1 }}>
				<Notifications position="bottom-right" />
				<App />
			</DatesProvider>
		</MantineProvider>
	</React.StrictMode>,
);

import {
	Combobox,
	createTheme,
	type MantineColorsTuple,
	Modal,
	Popover,
} from "@mantine/core";

// 画面で使う色は青 1 色にする。6 番目 (#2F4FC4) が基準の色。
const blue: MantineColorsTuple = [
	"#EEF1FB",
	"#DCE2F7",
	"#B6C3EE",
	"#8EA2E4",
	"#6D87DB",
	"#4E6BD1",
	"#2F4FC4",
	"#2843AA",
	"#1E3A9E",
	"#162C7A",
];

export const theme = createTheme({
	fontFamily:
		'system-ui, -apple-system, "Segoe UI", "Hiragino Sans", "Yu Gothic UI", "Meiryo", sans-serif',
	// 等幅の字体は、app.css の --mono と同じものにする。
	fontFamilyMonospace: "var(--mono)",
	defaultRadius: 6,
	primaryColor: "brand",
	primaryShade: 6,
	colors: { brand: blue },
	components: {
		// 候補の一覧や吹き出しは、下の内容 (プレビューなど) と同じ白なので、影と濃い枠で浮かせる。
		// Select や検索欄の一覧は Combobox を、カレンダーや誤りの吹き出しは Popover を使う。
		Combobox: Combobox.extend({
			defaultProps: { shadow: "md" },
			styles: { dropdown: { borderColor: "var(--border-strong)" } },
		}),
		// ダイアログは、どこから開いても同じ位置と見出しにする。
		Modal: Modal.extend({
			defaultProps: { centered: true },
			classNames: { title: "dialog-title" },
		}),
		Popover: Popover.extend({
			defaultProps: { shadow: "md" },
			styles: { dropdown: { borderColor: "var(--border-strong)" } },
		}),
	},
});

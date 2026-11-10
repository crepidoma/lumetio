import type { ReactNode } from "react";

// 線で描く小さなアイコン。色は文字の色 (currentColor) に合わせる。

type Props = { size?: number };

function Icon({ size = 16, children }: Props & { children: ReactNode }) {
	return (
		<svg
			width={size}
			height={size}
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			strokeWidth="2"
			strokeLinecap="round"
			strokeLinejoin="round"
			aria-hidden="true"
		>
			{children}
		</svg>
	);
}

export const ChevronDown = (p: Props) => (
	<Icon {...p}>
		<path d="M6 9l6 6 6-6" />
	</Icon>
);

export const ChevronLeft = (p: Props) => (
	<Icon {...p}>
		<path d="M15 18l-6-6 6-6" />
	</Icon>
);

export const ChevronRight = (p: Props) => (
	<Icon {...p}>
		<path d="M9 18l6-6-6-6" />
	</Icon>
);

export const Search = (p: Props) => (
	<Icon {...p}>
		<circle cx="11" cy="11" r="7" />
		<path d="M21 21l-4.3-4.3" />
	</Icon>
);

export const Plus = (p: Props) => (
	<Icon {...p}>
		<path d="M12 5v14M5 12h14" />
	</Icon>
);

export const Star = ({ filled, ...p }: Props & { filled?: boolean }) => (
	<Icon {...p}>
		<path
			d="M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z"
			fill={filled ? "currentColor" : "none"}
		/>
	</Icon>
);

export const ChevronsLeft = (p: Props) => (
	<Icon {...p}>
		<path d="M11 17l-5-5 5-5M18 17l-5-5 5-5" />
	</Icon>
);

export const ChevronsRight = (p: Props) => (
	<Icon {...p}>
		<path d="M13 17l5-5-5-5M6 17l5-5-5-5" />
	</Icon>
);

export const Grip = (p: Props) => (
	<Icon {...p}>
		<circle cx="9" cy="6" r="1" />
		<circle cx="15" cy="6" r="1" />
		<circle cx="9" cy="12" r="1" />
		<circle cx="15" cy="12" r="1" />
		<circle cx="9" cy="18" r="1" />
		<circle cx="15" cy="18" r="1" />
	</Icon>
);

export const Pencil = (p: Props) => (
	<Icon {...p}>
		<path d="M12 20h9" />
		<path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" />
	</Icon>
);

export const Check = (p: Props) => (
	<Icon {...p}>
		<path d="M20 6L9 17l-5-5" />
	</Icon>
);

export const List = (p: Props) => (
	<Icon {...p}>
		<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
	</Icon>
);

export const Gear = (p: Props) => (
	<Icon {...p}>
		<circle cx="12" cy="12" r="3" />
		<path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" />
	</Icon>
);

export const Folder = (p: Props) => (
	<Icon {...p}>
		<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
	</Icon>
);

export const Undo = (p: Props) => (
	<Icon {...p}>
		<path d="M3 12a9 9 0 1 0 2.6-6.4L3 8" />
		<path d="M3 3v5h5" />
	</Icon>
);

export const Trash = (p: Props) => (
	<Icon {...p}>
		<path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6" />
	</Icon>
);

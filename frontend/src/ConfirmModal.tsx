import { Button, Group, Modal, ModalStackContext } from "@mantine/core";
import { type ReactNode, use, useId } from "react";

// useStackedOverlay は、Modal.Stack の中で別のダイアログの上に重ねるときの幕の設定。
// 下のダイアログも自分の幕を出したままにする (Modal.Stack に入れる側で overlayProps の visible を渡す) ので、
// 重ねる側の幕は薄くする。本体は 2 枚分暗く、下のダイアログは 1 枚分だけ暗くなり、本体に溶け込まずに見分けられる。
// Modal.Stack の外で開くときは、いつもの幕にする。
export function useStackedOverlay() {
	return use(ModalStackContext) ? { backgroundOpacity: 0.25 } : undefined;
}

// ConfirmModal は、操作の前に確かめるダイアログ。右下に「キャンセル」と続ける操作を並べる。
// 問いかけ (title) は見出しにせず、本文と同じ大きさの文で出す。短い問いかけだけのダイアログで、文が浮いて見えないため。
// やめる操作は「キャンセル」があるので、右上の閉じるボタンは置かない。
// ダイアログの上に開くときは、そのダイアログと同じ Modal.Stack に入る (外なら何もしない)。
// Esc で閉じようとして確かめるとき、同じ Esc の押下で開いた直後のこのダイアログまで閉じないため。
export function ConfirmModal({
	opened,
	title,
	children,
	onCancel,
	actions,
}: {
	opened: boolean;
	title: string;
	children?: ReactNode;
	onCancel: () => void;
	// actions は「キャンセル」の右に並べるボタン。
	actions: ReactNode;
}) {
	const stackId = useId();
	const overlayProps = useStackedOverlay();
	return (
		<Modal
			opened={opened}
			onClose={onCancel}
			title={title}
			stackId={stackId}
			overlayProps={overlayProps}
			size={420}
			padding="lg"
			withCloseButton={false}
			classNames={{ header: "confirm-header", title: "confirm-message" }}
		>
			{children && <div className="confirm-detail">{children}</div>}
			<Group justify="flex-end" gap="xs" mt="lg">
				<Button variant="default" onClick={onCancel}>
					キャンセル
				</Button>
				{actions}
			</Group>
		</Modal>
	);
}

// DiscardConfirm は、保存していない変更があるまま閉じようとしたときに、破棄してよいかを確かめる。
export function DiscardConfirm({
	opened,
	onCancel,
	onDiscard,
}: {
	opened: boolean;
	onCancel: () => void;
	onDiscard: () => void;
}) {
	return (
		<ConfirmModal
			opened={opened}
			title="保存していない変更を破棄しますか？"
			onCancel={onCancel}
			actions={
				<Button color="red" onClick={onDiscard}>
					破棄する
				</Button>
			}
		/>
	);
}

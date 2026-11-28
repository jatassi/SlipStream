import type { ConversionsConversionGroup, ConversionsConversionRow } from "../api/gen/model";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { cancel, cancelGroup } from "./actions";
import { subjectOf } from "./row-parts";

type Open = { open: boolean; onOpenChange: (open: boolean) => void };

export const countWords = (count: number) =>
  `${count} ${count === 1 ? "Conversion" : "Conversions"}`;

export function CancelDialog({ row, ...open }: Open & { row: ConversionsConversionRow }) {
  return (
    <ConfirmDialog
      title="Cancel this Conversion?"
      confirmLabel="Cancel Conversion"
      dismissLabel="Keep Conversion"
      look="fail"
      {...open}
      onConfirm={() => cancel(row.id)}
    >
      SlipStream won&apos;t make the {row.target.name} file of {subjectOf(row)} until you ask for it
      again.
    </ConfirmDialog>
  );
}

export function CancelGroupDialog({
  group,
  ...open
}: Open & { group: ConversionsConversionGroup }) {
  return (
    <ConfirmDialog
      title={`Cancel ${countWords(group.count)}?`}
      confirmLabel="Cancel all"
      dismissLabel="Keep them"
      look="fail"
      {...open}
      onConfirm={() => cancelGroup(group.id)}
    >
      SlipStream won&apos;t make these files until you ask for them again.
    </ConfirmDialog>
  );
}

import type { ReactNode } from "react";

import { sentenceOf } from "../api/errors";
import type { MovesMovePreview, MovesStartedMoves } from "../api/gen/model";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";
import { toast } from "../ui/toast";
import { Plan } from "./preview-plan";
import { type Previewing, useMovePreview } from "./use-move-preview";

/** Toasts each unit that Start left out, then what started. */
export function told(started: MovesStartedMoves, startedWords: string) {
  for (const { name, words } of started.dropped) {
    toast(`${name} stays where it is. ${words}`, { kind: "fail" });
  }
  toast(startedWords);
}

export type PreviewDialogProps = {
  readonly title: string;
  readonly previewing: Previewing | undefined;
  readonly picker?: ReactNode;
  /** The Works without files a Start homes; a Move back homes none. */
  readonly homedWords?: (homed: number) => string;
  readonly startedWords: (started: number, plan: MovesMovePreview) => string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
};

/** A Move's preview and its Start, which names how many units move. */
export function PreviewDialog({
  title,
  previewing,
  picker,
  homedWords,
  startedWords,
  open,
  onOpenChange,
}: PreviewDialogProps) {
  const { preview, start } = useMovePreview(previewing);
  const plan = preview.data;
  return (
    <ConfirmDialog
      title={title}
      size={680}
      confirmLabel={plan?.button ?? "Move"}
      confirmDisabled={plan === undefined || plan.movable === 0 || preview.isFetching}
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={async () => {
        if (plan !== undefined) {
          const started = await start(plan);
          told(started, startedWords(started.started, plan));
        }
      }}
    >
      {picker}
      {preview.isFetching && <Spinner label="Checking what this Move needs" />}
      {preview.isError && (
        <SignalLine tone="fail" announce>
          {sentenceOf(preview.error)}
        </SignalLine>
      )}
      {plan !== undefined && (
        <Plan plan={plan} homed={plan.homed > 0 ? homedWords?.(plan.homed) : undefined} />
      )}
    </ConfirmDialog>
  );
}

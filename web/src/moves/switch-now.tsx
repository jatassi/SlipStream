import type { MovesMoveUnitWaiting } from "../api/gen/model";
import { switchMoveUnitNow } from "../api/gen/query/moves/moves";
import { ConfirmDialog } from "../ui/confirm-dialog";

const playback =
  "Someone may still be playing it. Their playback may stop when the old files are deleted.";
const watched = (name: string) =>
  `A Media Server that hasn't saved it yet loses who watched ${name}, their resume positions and ratings, and its collections.`;

/** What a Switch now gives up for a unit waiting for waiting, or for either wait when it isn't known. */
export function lostWords(name: string, waiting: MovesMoveUnitWaiting | undefined): string {
  if (waiting === "playing") {
    return playback;
  }
  if (waiting === "media_servers") {
    return watched(name);
  }
  return `${playback} ${watched(name)}`;
}

/** Asks before a Switch now, naming what it gives up. */
export function SwitchNowDialog({
  unitId,
  name,
  waiting,
  open,
  onOpenChange,
}: {
  unitId: string;
  name: string;
  waiting: MovesMoveUnitWaiting | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <ConfirmDialog
      title={`Switch ${name} now?`}
      confirmLabel="Switch now"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={() => switchMoveUnitNow(unitId)}
    >
      <p>{lostWords(name, waiting)}</p>
    </ConfirmDialog>
  );
}

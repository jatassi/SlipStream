import { useState } from "react";

import { isProblem } from "../../../../api/errors";
import type { NodesBoardNode, NodesRoleBoard } from "../../../../api/gen/model";
import { removeNode } from "../../../../api/gen/query/nodes/nodes";
import { blocks, heldWords, labelOf, refusedOutcome } from "../../../../nodes/roles";
import { Button } from "../../../../ui/button";
import { ConfirmDialog } from "../../../../ui/confirm-dialog";
import { Dialog } from "../../../../ui/dialog";

type Props = {
  readonly board: NodesRoleBoard;
  readonly seat: NodesBoardNode;
  readonly onClose: () => void;
  readonly onRemoved: () => Promise<void>;
};

const closing = (onClose: () => void) => (open: boolean) => {
  if (!open) {
    onClose();
  }
};

/** Why the Controller stays. */
export const controllerStays = (name: string): string =>
  `${name} is the Controller and keeps the library, so it can't be removed.`;

function ControllerStays({ seat, onClose }: Pick<Props, "seat" | "onClose">) {
  return (
    <Dialog title={`${seat.name} can't be removed`} open onOpenChange={closing(onClose)}>
      <p className="text-ink-2">{controllerStays(seat.name)}</p>
      <div className="mt-6 flex justify-end">
        <Button onClick={onClose}>Close</Button>
      </div>
    </Dialog>
  );
}

const heldLines = (board: NodesRoleBoard, seat: NodesBoardNode): string[] =>
  seat.roles.flatMap(({ role, held }) =>
    held === undefined
      ? []
      : [
          `${seat.name} still has ${heldWords(held)} for ${labelOf(board, role)}. Untick its Roles and remove ${seat.name} once that work finishes, or remove it now and give up that work.`,
        ],
  );

function RemoveWords({ name, holds }: { name: string; holds: readonly string[] }) {
  return (
    <>
      {holds.map((line) => (
        <p key={line} className="text-ink-1">
          {line}
        </p>
      ))}
      <p>
        {name} comes off the Nodes list, and its Roles move to other Nodes as if you unticked them.
        Past activity still shows its name, marked removed.
      </p>
      {holds.length > 0 && (
        <p>
          Its downloads that aren&apos;t imported yet fail, and SlipStream searches for them again.
          No files are deleted.
        </p>
      )}
      <p>
        {name} stops working with this Controller and keeps its files. To use it again, join it with
        a new Join Code.
      </p>
    </>
  );
}

/**
 * Remove a Node after re-confirming. Its Roles fall back as if unticked, and
 * work it still holds is written off, files kept, once the Admin says so.
 */
export function RemoveDialog({ board, seat, onClose, onRemoved }: Props) {
  const [refused, setRefused] = useState<readonly string[]>();
  if (seat.controller) {
    return <ControllerStays seat={seat} onClose={onClose} />;
  }
  const holds = refused ?? heldLines(board, seat);
  const writeOff = holds.length > 0;
  return (
    <ConfirmDialog
      title={`Remove ${seat.name}?`}
      confirmLabel={writeOff ? "Remove and give up its work" : "Remove"}
      look="fail"
      open
      onOpenChange={closing(onClose)}
      onConfirm={async () => {
        try {
          await removeNode(seat.id, { writeOff });
        } catch (error) {
          const outcome = refusedOutcome(error);
          if (!isProblem(error, "node_holds_work") || outcome === undefined) {
            throw error;
          }
          setRefused(blocks(outcome));
          throw new Error(`${seat.name} started holding work, so nothing was removed.`, {
            cause: error,
          });
        }
        await onRemoved();
      }}
    >
      <RemoveWords name={seat.name} holds={holds} />
    </ConfirmDialog>
  );
}

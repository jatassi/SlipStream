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

/** Why the Controller stays, and what retires it instead. */
export const controllerStays = (name: string): string =>
  `${name} is the Controller, which keeps the library, so it can't be removed. To retire it, hand the Controller over to another Node first. Handover comes in a later version of SlipStream.`;

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
          `${seat.name} still holds ${heldWords(held)} for ${labelOf(board, role)}. Untick its Roles and remove it when they finish, or remove it now and write them off.`,
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
        {name} leaves the Roster, and its Roles fall back as if you unticked them. History keeps its
        name, marked removed.
      </p>
      {holds.length > 0 && (
        <p>
          Writing off sends what isn&apos;t imported yet back to wanted, so SlipStream searches for
          it again. No files are deleted.
        </p>
      )}
      <p>
        {name} stops working for this SlipStream and keeps its files. To use it again, join it with
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
      confirmLabel={writeOff ? "Remove and write off" : "Remove"}
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

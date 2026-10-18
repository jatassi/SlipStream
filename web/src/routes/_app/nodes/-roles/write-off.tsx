import { useState } from "react";

import { sentenceOf } from "../../../../api/errors";
import type { NodesBoardNode, NodesHeld, NodesRoleBoard } from "../../../../api/gen/model";
import { writeOff } from "../../../../api/gen/query/nodes/nodes";
import {
  type Draft,
  type RoleName,
  blocks,
  heldWords,
  labelOf,
  refusedOutcome,
  withRole,
} from "../../../../nodes/roles";
import { Checkbox } from "../../../../ui/checkbox";
import { ConfirmDialog } from "../../../../ui/confirm-dialog";

type Holding = { readonly role: RoleName; readonly work: NodesHeld };

async function writeOffOrSay(nodeId: string, roles: readonly RoleName[]): Promise<void> {
  if (roles.length === 0) {
    throw new Error("Choose what to write off.");
  }
  try {
    await writeOff(nodeId, { roles: [...roles] });
  } catch (error) {
    const outcome = refusedOutcome(error);
    throw new Error(outcome === undefined ? sentenceOf(error) : blocks(outcome).join(" "), {
      cause: error,
    });
  }
}

function HeldList({
  board,
  seat,
  held,
  chosen,
  onChosen,
}: {
  board: NodesRoleBoard;
  seat: NodesBoardNode;
  held: readonly Holding[];
  chosen: Draft;
  onChosen: (chosen: Draft) => void;
}) {
  const [only] = held;
  if (held.length === 1 && only !== undefined) {
    return (
      <p className="text-ink-1">
        {seat.name} holds {heldWords(only.work)} for {labelOf(board, only.role)}.
      </p>
    );
  }
  return (
    <fieldset className="flex flex-col">
      <legend className="mb-1 text-ink-1">{seat.name} holds:</legend>
      {held.map(({ role, work }) => (
        <Checkbox
          key={role}
          label={`${heldWords(work)} for ${labelOf(board, role)}`}
          checked={chosen.has(role)}
          onCheckedChange={(on) => {
            onChosen(withRole(chosen, role, on));
          }}
        />
      ))}
    </fieldset>
  );
}

/**
 * Write off: give up what a Node holds for its Roles, after re-confirming,
 * with the count and size shown first.
 */
export function WriteOffDialog({
  board,
  seat,
  onClose,
  onWrittenOff,
}: {
  board: NodesRoleBoard;
  seat: NodesBoardNode;
  onClose: () => void;
  onWrittenOff: () => Promise<void>;
}) {
  const held = seat.roles.flatMap(({ role, held: work }) =>
    work === undefined ? [] : [{ role, work }],
  );
  const [chosen, setChosen] = useState<Draft>(() => new Set(held.map(({ role }) => role)));
  const roles = held.map(({ role }) => role).filter((role) => chosen.has(role));
  const unticks = roles.filter((role) => seat.assigned.includes(role));
  return (
    <ConfirmDialog
      title={`Write off what ${seat.name} holds`}
      confirmLabel="Write off"
      look="fail"
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onConfirm={async () => {
        await writeOffOrSay(seat.id, roles);
        await onWrittenOff();
      }}
    >
      <HeldList board={board} seat={seat} held={held} chosen={chosen} onChosen={setChosen} />
      <p>
        SlipStream stops waiting for it. What isn&apos;t imported yet goes back to wanted, so
        SlipStream searches for it again. No files are deleted.
      </p>
      {seat.away && <p>{seat.name} is offline, so it lets the work go when it comes back.</p>}
      {unticks.length > 0 && (
        <p>
          This also unticks {unticks.map((role) => labelOf(board, role)).join(" and ")} on{" "}
          {seat.name}.
        </p>
      )}
    </ConfirmDialog>
  );
}

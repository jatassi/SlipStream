import { useState } from "react";

import type { NodesBoardNode, NodesNode, NodesRoleBoard } from "../../../../api/gen/model";
import { type Draft, assignmentOf } from "../../../../nodes/roles";
import { useNow } from "../../../../tasks/use-now";
import { Button } from "../../../../ui/button";
import { useShell } from "../../../../ui/shell-context";
import { RoleGroups } from "../-roles/groups";
import type { Change } from "../-roles/plan";
import { ReviewDialog } from "../-roles/review";

export type PickProps = {
  readonly board: NodesRoleBoard;
  readonly seat: NodesBoardNode;
  readonly node: NodesNode;
  readonly onClose: () => void;
  readonly onApplied: () => Promise<void>;
};

function Actions({
  draft,
  onClose,
  onReview,
}: {
  draft: Draft;
  onClose: () => void;
  onReview: () => void;
}) {
  const stacked = useShell() === "phone";
  return (
    <div className={`flex gap-2 ${stacked ? "flex-col-reverse" : "justify-end"}`}>
      <Button onClick={onClose}>Not now</Button>
      <Button look="primary" disabled={draft.size === 0} onClick={onReview}>
        Review
      </Button>
    </div>
  );
}

/** The new Node's Role Groups, ticked from what it can do (its `suggested` Roles). */
export function Pick({ board, seat, node, onClose, onApplied }: PickProps) {
  const [draft, setDraft] = useState<Draft>(() => new Set(seat.suggested));
  const [reviewing, setReviewing] = useState<Change>();
  const now = useNow(60_000);
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">Choose what {node.name} does. It starts with what it can do.</p>
      <RoleGroups
        board={board}
        seat={seat}
        node={node}
        draft={draft}
        clock={{ now, zone: undefined }}
        onDraft={setDraft}
      />
      <Actions
        draft={draft}
        onClose={onClose}
        onReview={() => {
          setReviewing({ basis: board.basis, set: [assignmentOf(node.id, draft)] });
        }}
      />
      {reviewing !== undefined && (
        <ReviewDialog
          change={reviewing}
          onClose={() => {
            setReviewing(undefined);
          }}
          onApplied={onApplied}
        />
      )}
    </div>
  );
}

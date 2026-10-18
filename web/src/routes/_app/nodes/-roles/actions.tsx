import type { NodesBoardNode, NodesRoleBoard } from "../../../../api/gen/model";
import { labelOf } from "../../../../nodes/roles";
import { Button } from "../../../../ui/button";

export function Suggestion({
  board,
  seat,
  onTick,
}: {
  board: NodesRoleBoard;
  seat: NodesBoardNode;
  onTick: () => void;
}) {
  const labels = seat.suggested.map((role) => labelOf(board, role)).join(" and ");
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-card bg-fill px-4 py-3">
      <p className="flex-1 text-ink-2">
        {seat.name} holds no Roles yet. It can start with {labels}.
      </p>
      <Button onClick={onTick}>Tick {labels}</Button>
    </div>
  );
}

export function Actions({
  seat,
  changed,
  onReview,
  onDiscard,
  onWriteOff,
}: {
  seat: NodesBoardNode;
  changed: boolean;
  onReview: () => void;
  onDiscard: () => void;
  onWriteOff: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button look="primary" disabled={!changed} onClick={onReview}>
        Review
      </Button>
      {changed && (
        <Button look="ghost" onClick={onDiscard}>
          Discard changes
        </Button>
      )}
      {seat.roles.some(({ held }) => held !== undefined) && (
        <Button className="ml-auto" onClick={onWriteOff}>
          Write off what {seat.name} holds
        </Button>
      )}
    </div>
  );
}

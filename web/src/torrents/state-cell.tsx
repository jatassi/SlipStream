import { Link } from "@tanstack/react-router";

import { Mark } from "../ui/state-mark";
import { type Row, stateLine } from "./model";
import type { CellProps } from "./row";
import { torrentMark } from "./terms";

/** Where a moving torrent is going and how far it got, linking to its Move in Activity. */
function MoveTag({ row }: { row: Row }) {
  if (row.move === null) {
    return null;
  }
  return (
    <Link
      to="/activity"
      preload={false}
      className="max-w-full truncate rounded-item text-meta text-ink-2 underline hover:text-ink-1"
    >
      {row.move.words}
    </Link>
  );
}

export function StateCell({ row, live, names, clock }: CellProps) {
  return (
    <span className="flex min-w-0 flex-col items-start gap-0.5">
      <Mark mark={torrentMark(row)} />
      <span className="max-w-full truncate text-meta text-ink-3">
        {stateLine(row, live, { node: names.node(row.nodeId), clock })}
      </span>
      <MoveTag row={row} />
    </span>
  );
}

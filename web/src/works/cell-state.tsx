import type { ReactNode } from "react";

import type { FilesCell, FilesItemStatusStatus } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { LiveChip } from "../ui/live-chip";
import { MetPill } from "../ui/met-pill";
import { SignalChip } from "../ui/signal-chip";
import { nameOf } from "./paths";

const chips: { readonly [S in NonNullable<FilesItemStatusStatus>]: ReactNode } = {
  unreleased: <Chip>Unreleased</Chip>,
  missing: <Chip>Missing</Chip>,
  downloading: <LiveChip>Downloading</LiveChip>,
  failed: <SignalChip kind="fail">Failed</SignalChip>,
  upgradable: <Chip>Upgradable</Chip>,
  met: <MetPill>Met</MetPill>,
};

export const ItemStatus = ({ status }: { status: FilesItemStatusStatus }) =>
  status === null ? null : chips[status];

export function CellState({ cell }: { cell: FilesCell | undefined }) {
  if (cell === undefined) {
    return null;
  }
  if (!("file" in cell)) {
    return chips[cell.state];
  }
  return (
    <span className="flex min-w-0 flex-col items-start gap-1">
      {chips[cell.state]}
      <span className="max-w-full truncate text-meta text-ink-3">
        <Data>{nameOf(cell.file.path)}</Data>
      </span>
    </span>
  );
}

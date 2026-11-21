import { Tick02Icon } from "@hugeicons/core-free-icons";
import type { ReactNode } from "react";

import type { BackupsBackup, BackupsDestination } from "../../api/gen/model";
import { Icon } from "../../ui/icon";
import { SignalChip } from "../../ui/signal-chip";
import { Spinner } from "../../ui/spinner";
import { type Names, copyWords } from "./words";

const marks: { readonly [S in BackupsBackup["state"]]: ReactNode } = {
  verified: (
    <span className="inline-flex items-center gap-1 text-meta text-ink-2">
      <Icon icon={Tick02Icon} className="size-4 text-ok" />
      Checked
    </span>
  ),
  failed: <SignalChip kind="fail">Failed</SignalChip>,
  taking: (
    <span className="inline-flex items-center gap-1.5 text-meta text-ink-2">
      <Spinner className="size-3.5" />
      Backing up…
    </span>
  ),
  deleting: <span className="text-meta text-ink-3">Deleting…</span>,
};

export const StateMark = ({ backup }: { backup: BackupsBackup }) => marks[backup.state];

export function copiesLine(
  backup: BackupsBackup,
  destinations: readonly BackupsDestination[],
  names: Names,
) {
  const parts = backup.copies.flatMap((copy) => {
    const destination = destinations.find((each) => each.id === copy.destinationId);
    return destination === undefined
      ? []
      : [`${names(destination.nodeId)} ${copyWords[copy.state]}`];
  });
  const where = backup.local ? [] : ["not on this machine"];
  const all = [...where, ...parts];
  return all.length === 0 ? undefined : `Copies: ${all.join(", ")}`;
}

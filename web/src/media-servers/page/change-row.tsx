import type { ReactNode } from "react";

import type {
  MediaserversChangeView as Change,
  MediaserversChangeViewKind as Kind,
} from "../../api/gen/model";
import { agoWords } from "../../format";
import { Chip } from "../../ui/chip";
import { Data } from "../../ui/data";
import { ListRow } from "../../ui/list";
import { SignalChip } from "../../ui/signal-chip";
import { fileName } from "./paths";

/** Where a change stands, as the Admin reads it. A due change that has failed a send is failing. */
export type Standing = "waiting" | "sending" | "failing" | "sent" | "lost";

export function standingOf(change: Change): Standing {
  if (change.state === "sent") {
    return change.lost.length > 0 ? "lost" : "sent";
  }
  if (change.state === "sending") {
    return "sending";
  }
  return change.state === "due" && change.attempts > 0 ? "failing" : "waiting";
}

const chips: { readonly [S in Standing]: ReactNode } = {
  waiting: <Chip>Waiting</Chip>,
  sending: <Chip>Sending</Chip>,
  failing: <SignalChip kind="fail">Failing</SignalChip>,
  sent: <Chip>Sent</Chip>,
  lost: <SignalChip kind="warn">Lost data</SignalChip>,
};

const kindLabels: { readonly [K in Kind]: string } = {
  added: "New file",
  replaced: "Replaced file",
  removed: "Removed file",
  moved: "Rename",
  sidecar_added: "New Sidecar",
  sidecar_replaced: "Replaced Sidecar",
  sidecar_removed: "Removed Sidecar",
  library: "Refresh",
};

function ChangeDetail({ change }: { change: Change }) {
  return (
    <div className="flex flex-col gap-0.5 pt-1 pb-1 text-meta text-ink-2">
      <p className="text-pretty">{change.words}</p>
      {change.path !== "" && (
        <Data className="text-ink-3 [overflow-wrap:anywhere]">{change.path}</Data>
      )}
      {change.fromPath !== undefined && (
        <span className="text-ink-3">
          {"From "}
          <Data className="[overflow-wrap:anywhere]">{change.fromPath}</Data>
        </span>
      )}
    </div>
  );
}

/** One change: its file, what kind it is and when, where it stands, and what it did or waits for. */
export function ChangeRow({ change, now }: { change: Change; now: Date }) {
  return (
    <ListRow
      title={change.kind === "library" ? "Whole library" : fileName(change.path)}
      meta={`${kindLabels[change.kind]} · ${agoWords(change.sentAt ?? change.createdAt, now)}`}
      trailing={chips[standingOf(change)]}
      detail={<ChangeDetail change={change} />}
    />
  );
}

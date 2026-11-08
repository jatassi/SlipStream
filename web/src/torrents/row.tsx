import type { ReactNode } from "react";

import type { Clock } from "../format";
import type { LiveRow } from "../live/torrents";
import { Data } from "../ui/data";
import { Poster } from "../ui/poster";
import { SignalChip } from "../ui/signal-chip";
import { type Row, rateWords, stateLine, stateWords, trackerLine } from "./model";
import type { Names } from "./names";

export { ProgressCell } from "./progress-cell";

export type CellProps = {
  readonly row: Row;
  readonly live: LiveRow | undefined;
  readonly names: Names;
  readonly clock: Clock;
};

function Lines({ first, second }: { first: ReactNode; second: ReactNode }) {
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <span className="truncate">{first}</span>
      <span className="truncate text-meta text-ink-3">{second}</span>
    </span>
  );
}

function WorkLines({ row, names }: CellProps) {
  const work = names.work(row.workId);
  if (work === undefined) {
    return <Lines first={<Data>{row.name}</Data>} second="Not in the Library" />;
  }
  const facts = [
    row.edition === undefined ? undefined : names.edition(row.edition),
    row.quality === undefined ? undefined : names.quality(row.quality),
  ].filter((fact) => fact !== undefined);
  return (
    <Lines
      first={<span className="font-semibold">{work}</span>}
      second={facts.length === 0 ? <Data>{row.name}</Data> : facts.join(" · ")}
    />
  );
}

export function WorkCell(props: CellProps) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <Poster url={props.names.poster(props.row.workId)} />
      <WorkLines {...props} />
    </span>
  );
}

const signalOf = { none: undefined, warning: "warn", failure: "fail" } as const;

export function StateCell({ row, live, names, clock }: CellProps) {
  const signal = signalOf[row.severity];
  const word = stateWords[row.state];
  return (
    <span className="flex min-w-0 flex-col items-start gap-0.5">
      {signal === undefined ? (
        <span className="truncate">{word}</span>
      ) : (
        <SignalChip kind={signal}>{word}</SignalChip>
      )}
      <span className="max-w-full truncate text-meta text-ink-3">
        {stateLine(row, live, { node: names.node(row.nodeId), clock })}
      </span>
    </span>
  );
}

export function SpeedCell({ live }: CellProps) {
  const down = rateWords(live?.downBps ?? 0);
  const up = rateWords(live?.upBps ?? 0);
  return (
    <span className="flex flex-col gap-0.5 text-meta">
      <Data className="text-ink-1">
        <span aria-label="Down">↓</span> {down}
      </Data>
      <Data className="text-ink-3">
        <span aria-label="Up">↑</span> {up}
      </Data>
    </span>
  );
}

export function TrackerCell({ row, clock }: CellProps) {
  const { host, status } = trackerLine(row.announce, clock);
  return <Lines first={host} second={status} />;
}

export function NodeCell({ row, names }: CellProps) {
  return <Lines first={names.node(row.nodeId)} second="Built-in" />;
}

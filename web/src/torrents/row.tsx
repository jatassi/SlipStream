import type { ReactNode } from "react";

import type { Clock } from "../format";
import type { LiveRow } from "../live/torrents";
import { Data } from "../ui/data";
import { LiveBar } from "../ui/live-bar";
import { Meter } from "../ui/meter";
import { SignalChip } from "../ui/signal-chip";
import {
  type Row,
  paceOf,
  percentOf,
  progressOf,
  rateWords,
  seedWords,
  stateLine,
  stateWords,
  trackerLine,
} from "./model";
import type { Names } from "./names";

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

export function WorkCell({ row, names }: CellProps) {
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

const moving = new Set<Row["state"]>(["downloading", "checking"]);

export function ProgressCell({ row, live }: CellProps) {
  const progress = progressOf(row, live);
  if (progress.done >= progress.wanted) {
    return (
      <span className="truncate text-meta text-ink-2">
        {seedWords(row)}
        <span data-column-echo="">
          {" · Ratio "}
          <Data>{row.ratio.toFixed(2)}</Data>
        </span>
      </span>
    );
  }
  const percent = percentOf(progress);
  const label = `${stateWords[row.state]} ${row.name}`;
  const { eta, rate } = paceOf(row.state, live);
  return (
    <span className="flex min-w-0 flex-col gap-1.5">
      <span className="flex items-baseline justify-between gap-3 text-meta text-ink-3">
        <Data className="text-ink-1">{percent}%</Data>
        <Data className="truncate">
          {eta}
          {rate !== "" && (
            <span data-column-echo="">
              {eta === "" ? "" : " · "}
              {rate}
            </span>
          )}
        </Data>
      </span>
      {moving.has(row.state) && row.lastKnown === undefined ? (
        <LiveBar label={label} value={percent} />
      ) : (
        <Meter label={label} value={percent} />
      )}
    </span>
  );
}

export function RatioCell({ row }: CellProps) {
  return <Data>{row.ratio.toFixed(2)}</Data>;
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
  const { host, status } = trackerLine(row.tracker, clock);
  return <Lines first={host} second={status} />;
}

export function NodeCell({ row, names }: CellProps) {
  return <Lines first={names.node(row.nodeId)} second="Built-in" />;
}

import type { ReactNode } from "react";

import type { Clock } from "../ui/clock";
import type { LiveRow } from "../live/torrents";
import { QualityName } from "../release/quality";
import { Data } from "../ui/data";
import { Artwork } from "../ui/artwork";
import { type Row, rateWords, trackerLine } from "./model";
import type { Names } from "./names";

export { ProgressCell } from "./progress-cell";
export { StateCell } from "./state-cell";

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
  const edition = row.edition === undefined ? undefined : names.edition(row.edition);
  return (
    <Lines
      first={<span className="font-semibold">{work}</span>}
      second={
        edition === undefined && row.quality === undefined ? (
          <Data>{row.name}</Data>
        ) : (
          <>
            {edition}
            {edition !== undefined && row.quality !== undefined && " · "}
            {row.quality !== undefined && <QualityName quality={row.quality} />}
          </>
        )
      }
    />
  );
}

export function WorkCell(props: CellProps) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <Artwork use="poster" size="row" url={props.names.poster(props.row.workId)} />
      <WorkLines {...props} />
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
  return <Lines first={names.node(row.nodeId)} second={row.clientName ?? "Built-in"} />;
}

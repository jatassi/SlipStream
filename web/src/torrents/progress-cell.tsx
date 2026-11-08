import { Data } from "../ui/data";
import { LaneMeter } from "../ui/lane-meter";
import { LiveBar } from "../ui/live-bar";
import { Meter } from "../ui/meter";
import { SignalChip } from "../ui/signal-chip";
import { type Row, paceOf, percentOf, progressOf, seedWords, stateWords } from "./model";
import type { CellProps } from "./row";
import { lanesOf, seedLine } from "./seeding";

function SeedingCell({ seeding }: { seeding: NonNullable<Row["seeding"]> }) {
  const risk = seeding.atRisk === "yes";
  const line = seedLine(seeding);
  return (
    <span className="flex min-w-0 flex-col gap-1.5">
      {(seeding.goalMet || risk || line !== "") && (
        <span className="flex min-w-0 items-center gap-2 text-meta text-ink-3">
          {seeding.goalMet && <SignalChip kind="ok">Goal met</SignalChip>}
          {risk && <SignalChip kind="warn">At risk</SignalChip>}
          <span className="truncate">{line}</span>
        </span>
      )}
      <LaneMeter lanes={lanesOf(seeding)} risk={risk} />
    </span>
  );
}

const moving = new Set<Row["state"]>(["downloading", "checking"]);

export function ProgressCell({ row, live }: CellProps) {
  const progress = progressOf(row, live);
  if (progress.done >= progress.wanted) {
    return row.seeding === null ? (
      <span className="truncate text-meta text-ink-2">
        {seedWords(row)}
        <span data-column-echo="">
          {" · Ratio "}
          <Data>{row.ratio.toFixed(2)}</Data>
        </span>
      </span>
    ) : (
      <SeedingCell seeding={row.seeding} />
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

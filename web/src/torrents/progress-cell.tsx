import { Data } from "../ui/data";
import { LaneMeter } from "../ui/lane-meter";
import { ProgressCell as Cell, percentOf } from "../ui/progress";
import { SignalChip } from "../ui/signal-chip";
import { type Row, paceOf, progressOf, seedWords } from "./model";
import type { CellProps } from "./row";
import { lanesOf, seedLine } from "./seeding";
import { torrentTerms } from "./terms";

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
  const { eta, rate } = paceOf(row.state, live);
  return (
    <Cell
      label={`${torrentTerms[row.state].word} ${row.name}`}
      value={percentOf(progress.done, progress.wanted, 100)}
      motion={moving.has(row.state) && row.lastKnown === undefined ? "live" : "settled"}
      span="fill"
      note={
        <Data>
          {eta}
          {rate !== "" && (
            <span data-column-echo="">
              {eta === "" ? "" : " · "}
              {rate}
            </span>
          )}
        </Data>
      }
    />
  );
}

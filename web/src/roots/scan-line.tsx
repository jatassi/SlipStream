import type { RootsRunning, RootsScan } from "../api/gen/model";
import { ProgressCell, percentOf } from "../ui/progress";
import { RelativeTime } from "../ui/relative-time";
import { filesWords } from "./refused";

const counted = new Intl.NumberFormat("en");

function runningWords({ phase, done, total }: RootsRunning): string {
  if (total === null) {
    return `Listing files, ${counted.format(done)} so far`;
  }
  return phase === "probing"
    ? `Reading media info, ${counted.format(done)} of ${filesWords(total)}`
    : `Comparing ${filesWords(total)}`;
}

function Running({ running }: { running: RootsRunning }) {
  const words = runningWords(running);
  if (running.phase !== "probing" || running.total === null) {
    return <span className="text-meta text-ink-2">{words}</span>;
  }
  return (
    <ProgressCell
      label={words}
      value={percentOf(running.done, running.total, 0)}
      motion="live"
      span="line"
      amount={words}
    />
  );
}

function Scanned({ at, summary }: { at: string; summary: string }) {
  return (
    <span className="text-meta text-ink-3">
      Scanned <RelativeTime at={at} form="age" />. {summary}
    </span>
  );
}

/** A root's library scan: its step while it runs, why it stopped, or when it last ran and what it found. */
export function ScanLine({ scan }: { scan: RootsScan }) {
  if (scan.running !== null) {
    return <Running running={scan.running} />;
  }
  if (scan.stopped !== null) {
    return <span className="text-meta text-warn-ink">{scan.summary}</span>;
  }
  return scan.lastAt === null ? (
    <span className="text-meta text-ink-3">Not scanned yet.</span>
  ) : (
    <Scanned at={scan.lastAt} summary={scan.summary} />
  );
}

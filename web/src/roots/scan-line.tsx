import type { RootsRunning, RootsScan } from "../api/gen/model";
import { agoWords } from "../format";
import { useNow } from "../tasks/use-now";
import { Meter } from "../ui/meter";
import { filesWords } from "./refused";

const counted = new Intl.NumberFormat("en");
const agoTick = 60_000;

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
  return (
    <span className="flex max-w-80 flex-col gap-1 text-meta text-ink-2">
      {words}
      {running.phase === "probing" && running.total !== null && (
        <Meter label={words} value={running.done} max={running.total} />
      )}
    </span>
  );
}

function Scanned({ at, summary }: { at: string; summary: string }) {
  const now = useNow(agoTick);
  return <span className="text-meta text-ink-3">{`Scanned ${agoWords(at, now)}. ${summary}`}</span>;
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

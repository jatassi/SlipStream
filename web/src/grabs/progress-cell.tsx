import { Link } from "@tanstack/react-router";

import type { GrabsFetchWait, GrabsGrab, GrabsGrabImport } from "../api/gen/model";
import { bytes } from "../format";
import { useLiveRow } from "../live/torrents";
import { type Row, paceOf } from "../torrents/model";
import { Data } from "../ui/data";
import { ProgressCell as Cell, percentOf } from "../ui/progress";
import { MetadataCell } from "./metadata-cell";
import { movesBytes, stageWord } from "./stages";

function Progress({
  grab,
  progress,
  pace,
}: {
  grab: GrabsGrab;
  progress: { done: number; total: number };
  pace: string;
}) {
  return (
    <Cell
      label={`${stageWord[grab.stage]} ${grab.releaseTitle}`}
      value={percentOf(progress.done, progress.total, 0)}
      motion="live"
      note={
        pace === "" ? (
          <>
            <Data>{bytes(progress.done)}</Data> of <Data>{bytes(progress.total)}</Data>
          </>
        ) : (
          <Data>{pace}</Data>
        )
      }
    />
  );
}

// A download's speed and time left come from its Torrent's live samples.
function LiveProgress({
  grab,
  torrentId,
  torrent,
}: {
  grab: GrabsGrab;
  torrentId: string;
  torrent: Row | undefined;
}) {
  const live = useLiveRow(torrentId);
  if (live === undefined) {
    return <ReportedProgress grab={grab} />;
  }
  const { eta, rate } = paceOf(torrent?.state, live);
  const pace = [eta, rate].filter((part) => part !== "").join(" · ");
  return <Progress grab={grab} progress={{ done: live.done, total: live.wanted }} pace={pace} />;
}

function ReportedProgress({ grab }: { grab: GrabsGrab }) {
  return (
    <Progress
      grab={grab}
      progress={grab.progress ?? { done: 0, total: grab.releaseSize }}
      pace=""
    />
  );
}

function FetchWaitCell({ wait }: { wait: GrabsFetchWait }) {
  return <span className="block max-w-[240px] text-meta text-ink-3">{wait.sentence}</span>;
}

/** "Importing 6 of 10 episodes", and while a file waits for the Admin, the link to its Pending import. */
function ImportCell({ progress }: { progress: GrabsGrabImport }) {
  return (
    <span className="flex max-w-[240px] flex-col gap-0.5 text-meta">
      <span className="text-ink-2">{progress.words}</span>
      {progress.waiting > 0 && (
        <Link
          to="/library/pending/$importId"
          params={{ importId: progress.id }}
          className="w-fit rounded-item text-ink-1 underline hover:text-ink-2"
        >
          Waiting for you
        </Link>
      )}
    </span>
  );
}

export function ProgressCell({ grab, torrent }: { grab: GrabsGrab; torrent: Row | undefined }) {
  if (grab.fetchWait !== undefined) {
    return <FetchWaitCell wait={grab.fetchWait} />;
  }
  if (grab.metadata !== undefined) {
    return <MetadataCell search={grab.metadata} />;
  }
  if (grab.stage === "importing" && grab.import !== null) {
    return <ImportCell progress={grab.import} />;
  }
  if (!movesBytes(grab)) {
    return null;
  }
  return grab.torrentId === null ? (
    <ReportedProgress grab={grab} />
  ) : (
    <LiveProgress grab={grab} torrentId={grab.torrentId} torrent={torrent} />
  );
}

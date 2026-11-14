import { Link } from "@tanstack/react-router";

import type { GrabsFetchWait, GrabsGrab, GrabsGrabImport } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { bytes } from "../format";
import { useLiveRow } from "../live/torrents";
import { paceOf } from "../torrents/model";
import { Data } from "../ui/data";
import { LiveBar } from "../ui/live-bar";
import { MetadataCell } from "./metadata-cell";
import { movesBytes, stageWord } from "./stages";

const percent = ({ done, total }: { done: number; total: number }) =>
  total === 0 ? 0 : Math.floor((done / total) * 100);

function Progress({
  grab,
  progress,
  pace,
}: {
  grab: GrabsGrab;
  progress: { done: number; total: number };
  pace: string;
}) {
  const shown = percent(progress);
  return (
    <span className="flex max-w-[240px] flex-col gap-1.5">
      <span className="flex items-baseline justify-between gap-3 text-meta text-ink-3">
        <Data className="text-ink-1">{shown}%</Data>
        {pace === "" ? (
          <span>
            <Data>{bytes(progress.done)}</Data> of <Data>{bytes(progress.total)}</Data>
          </span>
        ) : (
          <Data className="truncate">{pace}</Data>
        )}
      </span>
      <LiveBar label={`${stageWord[grab.stage]} ${grab.releaseTitle}`} value={shown} />
    </span>
  );
}

// A download's speed and time left come from its Torrent's live samples.
function LiveProgress({ grab, torrentId }: { grab: GrabsGrab; torrentId: string }) {
  const live = useLiveRow(torrentId);
  const state = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === torrentId)?.state },
  }).data;
  if (live === undefined) {
    return <ReportedProgress grab={grab} />;
  }
  const { eta, rate } = paceOf(state, live);
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

export function ProgressCell({ grab }: { grab: GrabsGrab }) {
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
    <LiveProgress grab={grab} torrentId={grab.torrentId} />
  );
}

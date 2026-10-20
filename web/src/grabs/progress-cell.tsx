import type { GrabsGrab } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { bytes } from "../format";
import { useLiveRow } from "../live/torrents";
import { paceOf } from "../torrents/model";
import { Data } from "../ui/data";
import { LiveBar } from "../ui/live-bar";
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

export function ProgressCell({ grab }: { grab: GrabsGrab }) {
  if (!movesBytes(grab)) {
    return null;
  }
  return grab.stage === "downloading" && grab.torrentId !== null ? (
    <LiveProgress grab={grab} torrentId={grab.torrentId} />
  ) : (
    <ReportedProgress grab={grab} />
  );
}

import type { GrabsGrab, GrabsMetadataSearch } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { bytes, countWords, sinceWords } from "../format";
import { useLiveRow } from "../live/torrents";
import { useNow } from "../tasks/use-now";
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

const peer = { one: "peer", many: "peers" };

function MetadataCell({ search }: { search: GrabsMetadataSearch }) {
  const reading = { now: useNow(60_000), zone: undefined };
  return (
    <span className="flex max-w-[240px] flex-col gap-0.5">
      <span>
        {search.peers === 0
          ? "Looking for peers"
          : `Getting metadata from ${countWords(search.peers, peer)}`}
      </span>
      <span className="text-meta text-ink-3">
        Fails at <Data>{sinceWords(search.endsAt, reading)}</Data> if no peer sends the metadata
      </span>
    </span>
  );
}

export function ProgressCell({ grab }: { grab: GrabsGrab }) {
  if (grab.metadata !== undefined) {
    return <MetadataCell search={grab.metadata} />;
  }
  if (!movesBytes(grab)) {
    return null;
  }
  return grab.stage === "downloading" && grab.torrentId !== null ? (
    <LiveProgress grab={grab} torrentId={grab.torrentId} />
  ) : (
    <ReportedProgress grab={grab} />
  );
}

import type { DownloadclientsTorrentDetail, DownloadclientsTrackerDetail } from "../api/gen/model";
import { type Clock, bytes, sinceWords } from "../format";
import type { LiveRow } from "../live/torrents";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { InspectorSection } from "../ui/inspector";
import { type Facts, FactList, lengthOf } from "./inspector-facts";
import { type Row, etaWords, percentOf, progressOf, rateWords, trackerLine } from "./model";

export function Numbers({
  row,
  detail,
  live,
}: {
  row: Row;
  detail: DownloadclientsTorrentDetail;
  live: LiveRow | undefined;
}) {
  const progress = progressOf(row, live);
  const eta = row.state === "downloading" ? etaWords(live) : "";
  const facts: Facts = [
    ["Progress", <Data key="p">{`${percentOf(progress)}% of ${bytes(progress.wanted)}`}</Data>],
    ...(eta === "" ? [] : ([["Time left", <Data key="t">{eta}</Data>]] as const)),
    ...(live === undefined
      ? []
      : ([
          [
            "Speed",
            <Data key="s">{`↓ ${rateWords(live.downBps)} · ↑ ${rateWords(live.upBps)}`}</Data>,
          ],
          ["Peers", `${live.peers} ${live.peers === 1 ? "peer" : "peers"}, ${live.seeds} seeding`],
        ] as const)),
    ["Downloaded", <Data key="d">{bytes(detail.downloaded)}</Data>],
    ["Uploaded", <Data key="u">{bytes(detail.uploaded)}</Data>],
    ["Ratio", <Data key="r">{detail.ratio.toFixed(2)}</Data>],
    ["Seed time", lengthOf(detail.seedingSeconds)],
    ["Active time", lengthOf(detail.activeSeconds)],
  ];
  return (
    <InspectorSection title="Numbers">
      <FactList facts={facts} />
    </InspectorSection>
  );
}

const hostOf = (url: string) => (URL.canParse(url) ? new URL(url).host : url);

function TrackerLine({ tracker, clock }: { tracker: DownloadclientsTrackerDetail; clock: Clock }) {
  const host = hostOf(tracker.url);
  const { status } = trackerLine({ host, ...tracker }, clock);
  const timing = [
    tracker.answeredAt === undefined
      ? "No answer yet"
      : `Answered at ${sinceWords(tracker.answeredAt, clock)}`,
    ...(tracker.nextAnnounce === undefined
      ? []
      : [`next announce at ${sinceWords(tracker.nextAnnounce, clock)}`]),
  ].join(", ");
  return (
    <li className="flex flex-col gap-0.5 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <span className="flex items-baseline justify-between gap-3">
        <span className="truncate font-semibold">{host}</span>
        <span className="shrink-0 text-ink-2">{status}</span>
      </span>
      <Details summary={<span className="text-meta text-ink-3">{timing}</span>}>
        <FactList
          facts={[
            ["Address", <Data key="a">{tracker.url}</Data>],
            ["Tier", String(tracker.tier + 1)],
            ...(tracker.minIntervalSeconds === undefined
              ? []
              : ([["Min interval", lengthOf(tracker.minIntervalSeconds)]] as const)),
            ...(tracker.acceptsAt === undefined
              ? []
              : ([["Accepts an announce", sinceWords(tracker.acceptsAt, clock)]] as const)),
          ]}
        />
      </Details>
    </li>
  );
}

export function Trackers({
  detail,
  node,
  clock,
}: {
  detail: DownloadclientsTorrentDetail;
  node: string;
  clock: Clock;
}) {
  const waits =
    detail.announceAt !== undefined && Date.parse(detail.announceAt) > clock.now.getTime();
  return (
    <InspectorSection title="Trackers">
      {!detail.live && (
        <p className="mb-2 text-ink-3">{`${node} isn't answering, so these are as it last reported.`}</p>
      )}
      {detail.trackers.length === 0 ? (
        <p className="text-ink-3">Its metainfo names no tracker.</p>
      ) : (
        <ul>
          {detail.trackers.map((tracker) => (
            <TrackerLine key={tracker.url} tracker={tracker} clock={clock} />
          ))}
        </ul>
      )}
      {waits && detail.announceAt !== undefined && (
        <p className="mt-2 text-ink-3">
          Announce now is offered from {sinceWords(detail.announceAt, clock)}, when every tracker
          accepts one.
        </p>
      )}
    </InspectorSection>
  );
}

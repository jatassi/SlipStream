import type { DownloadclientsTorrentDetail, DownloadclientsTrackerDetail } from "../api/gen/model";
import { bytes } from "../format";
import type { LiveRow } from "../live/torrents";
import type { Clock } from "../ui/clock";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { InspectorSection } from "../ui/inspector";
import { percentOf } from "../ui/progress";
import { RelativeTime } from "../ui/relative-time";
// oxlint-disable-next-line import/max-dependencies -- the numbers read the torrent, its trackers and the clock
import { whenWords } from "../ui/time-words";
import { type Row, paceOf, progressOf, rateWords, seedLength, trackerLine } from "./model";

const lengthOf = (seconds: number) => (seconds < 60 ? "Under a minute" : seedLength(seconds));

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
  const { eta } = paceOf(row.state, live);
  return (
    <InspectorSection title="Numbers">
      {row.clientName !== undefined && (
        <p className="mb-3 text-ink-2">
          {`${row.clientName}'s own seeding rules apply to this torrent.`}
        </p>
      )}
      <Facts look="inline">
        <Fact label="Progress">
          <Data>{`${percentOf(progress.done, progress.wanted, 100)}% of ${bytes(progress.wanted)}`}</Data>
        </Fact>
        {eta !== "" && (
          <Fact label="Time left">
            <Data>{eta}</Data>
          </Fact>
        )}
        {live !== undefined && (
          <>
            <Fact label="Speed">
              <Data>{`↓ ${rateWords(live.downBps)} · ↑ ${rateWords(live.upBps)}`}</Data>
            </Fact>
            <Fact label="Peers">
              {`${live.peers} ${live.peers === 1 ? "peer" : "peers"}, ${live.seeds} seeding`}
            </Fact>
          </>
        )}
        <Fact label="Downloaded">
          <Data>{bytes(detail.downloaded)}</Data>
        </Fact>
        <Fact label="Uploaded">
          <Data>{bytes(detail.uploaded)}</Data>
        </Fact>
        <Fact label="Active time">{lengthOf(detail.activeSeconds)}</Fact>
      </Facts>
    </InspectorSection>
  );
}

const hostOf = (url: string) => (URL.canParse(url) ? new URL(url).host : url);

const counted = (count: number | undefined, one: string, many: string) =>
  count === undefined ? [] : [`${count} ${count === 1 ? one : many}`];

/** "142 seeders and 3 leechers on seedpool.", or undefined when the response carried no counts. */
function swarmSentence({ seeders, leechers }: DownloadclientsTrackerDetail, on: string) {
  const counts = [
    ...counted(seeders, "seeder", "seeders"),
    ...counted(leechers, "leecher", "leechers"),
  ];
  return counts.length === 0 ? undefined : `${counts.join(" and ")} on ${on}.`;
}

function TrackerLine({
  tracker,
  named,
  clock,
}: {
  tracker: DownloadclientsTrackerDetail;
  named: string | undefined;
  clock: Clock;
}) {
  const host = hostOf(tracker.url);
  const swarm = swarmSentence(tracker, named ?? host);
  const { status } = trackerLine({ host, ...tracker }, clock);
  const timing = [
    tracker.answeredAt === undefined
      ? "No response yet"
      : `Responded ${whenWords(tracker.answeredAt, "at", clock)}`,
    ...(tracker.nextAnnounce === undefined
      ? []
      : [`next announce ${whenWords(tracker.nextAnnounce, "at", clock)}`]),
  ].join(", ");
  return (
    <li className="flex flex-col gap-0.5 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]">
      <span className="flex items-baseline justify-between gap-3">
        <span className="truncate font-semibold">{host}</span>
        <span className="shrink-0 text-ink-2">{status}</span>
      </span>
      {swarm !== undefined && <span className="text-ink-2 tabular-nums">{swarm}</span>}
      <Details summary={<span className="text-meta text-ink-3">{timing}</span>}>
        <Facts look="well">
          <Fact label="Address">
            <Data>{tracker.url}</Data>
          </Fact>
          <Fact label="Tier">{tracker.tier + 1}</Fact>
          {tracker.minIntervalSeconds !== undefined && (
            <Fact label="Min interval">{lengthOf(tracker.minIntervalSeconds)}</Fact>
          )}
          {tracker.acceptsAt !== undefined && (
            <Fact label="Accepts an announce">
              <RelativeTime at={tracker.acceptsAt} form="clock" />
            </Fact>
          )}
        </Facts>
      </Details>
    </li>
  );
}

export function AnnounceUrls({
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
        <p className="mb-2 text-ink-3">{`${node} isn't responding, so these are from its last report.`}</p>
      )}
      {detail.trackers.length === 0 ? (
        <p className="text-ink-3">This torrent lists no trackers.</p>
      ) : (
        <ul>
          {detail.trackers.map((tracker) => (
            <TrackerLine
              key={tracker.url}
              tracker={tracker}
              named={detail.private ? detail.tracker?.name : undefined}
              clock={clock}
            />
          ))}
        </ul>
      )}
      {waits && detail.announceAt !== undefined && (
        <p className="mt-2 text-ink-3">
          Announce now is offered {whenWords(detail.announceAt, "from", clock)}, when every tracker
          allows another announce.
        </p>
      )}
    </InspectorSection>
  );
}

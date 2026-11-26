import type {
  MediaserversServerView as Server,
  PlaybackReport as Report,
} from "../../api/gen/model";
import { useGetMediaServerPlaybackSuspense } from "../../api/gen/query/mediaservers/mediaservers";
import { useListContentTypesSuspense } from "../../api/gen/query/works/works";
import { countWords, durationWords } from "../../format";
import { type Clock, useClock } from "../../ui/clock";
import { Empty } from "../../ui/empty";
import { List } from "../../ui/list";
import { SignalLine } from "../../ui/signal-line";
import { whenWords } from "../../ui/time-words";
import { PlayerRow } from "./player-row";

const plays = { one: "play", many: "plays" };

/** The report's window in a sentence: the setting's length, or where the history starts when that is later. */
function windowWords(report: Report, clock: Clock): string {
  const full = clock.now.getTime() - report.window * 1000;
  return new Date(report.since).getTime() > full
    ? whenWords(report.since, "since", clock)
    : `in the last ${durationWords(report.window)}`;
}

function serverLine(server: string, report: Report, window: string): string {
  return report.transcoded === 0
    ? `None of the ${countWords(report.plays, plays)} on ${server} ${window} ${report.plays === 1 ? "was" : "were"} transcoded.`
    : `${server} transcoded ${report.transcoded} of ${countWords(report.plays, plays)} ${window}.`;
}

/** Which players' plays the server transcoded and why, worst first, each with what would have avoided it. */
export function PlaybackTab({ server }: { server: Server }) {
  const report = useGetMediaServerPlaybackSuspense(server.id).data;
  const types = useListContentTypesSuspense().data.contentTypes;
  const clock = useClock();
  const window = windowWords(report, clock);
  const since = whenWords(report.since, "since", clock);
  const node = server.reachingNode.name;
  return (
    <div className="flex flex-col gap-4">
      {!report.reachable && (
        <SignalLine tone="warn">
          {`SlipStream doesn't record plays on ${server.name} while ${node} is offline. Recording starts again when ${node} is back.`}
        </SignalLine>
      )}
      {report.plays === 0 ? (
        <Empty look="sheet">{`Nothing has played on ${server.name} ${window}.`}</Empty>
      ) : (
        <>
          <p className="text-pretty text-ink-1">{serverLine(server.name, report, window)}</p>
          <List label={`Players on ${server.name}`}>
            {report.players.map((player) => (
              <PlayerRow
                key={player.key}
                ctx={{
                  server: server.name,
                  player: player.name,
                  window,
                  since,
                  plex: report.plex,
                  types,
                }}
                serverId={server.id}
                report={report}
                player={player}
              />
            ))}
          </List>
        </>
      )}
    </div>
  );
}

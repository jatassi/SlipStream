import type { MediaserversServerView as Server } from "../../api/gen/model";
import { useListMediaServerSessionsSuspense } from "../../api/gen/query/mediaservers/mediaservers";
import { Empty } from "../../ui/empty";
import { List } from "../../ui/list";
import { SessionRow } from "./session-row";

/** Who is playing what on the server right now, and whether it plays the file as it is. */
export function PlayingTab({ server }: { server: Server }) {
  const { reachable, sessions } = useListMediaServerSessionsSuspense(server.id).data;
  if (!reachable) {
    return (
      <Empty look="sheet">
        {`SlipStream can't check what's playing on ${server.name} while ${server.reachingNode.name} is offline.`}
      </Empty>
    );
  }
  if (sessions.length === 0) {
    return <Empty look="sheet">{`Nobody is playing anything on ${server.name} right now.`}</Empty>;
  }
  return (
    <List label={`Now playing on ${server.name}`}>
      {sessions.map((session) => (
        <SessionRow key={session.playId} server={server} session={session} />
      ))}
    </List>
  );
}

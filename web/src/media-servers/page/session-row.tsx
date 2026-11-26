import type {
  MediaserversServerView as Server,
  MediaserversSessionView as Session,
} from "../../api/gen/model";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { ListRow } from "../../ui/list";
import { PlainTime } from "../../ui/relative-time";
import { Mark } from "../../ui/state-mark";
import { fileName } from "./paths";
import { decisionTerm, nowPlayingLine } from "./playback-words";

function SessionDetail({ server, session }: { server: Server; session: Session }) {
  const line = nowPlayingLine(server.name, session);
  return (
    <div className="flex flex-col gap-1 pt-1 pb-1 text-meta text-ink-2">
      {line !== undefined && <p className="text-pretty">{line}</p>}
      <Details>
        <div className="flex flex-col gap-1 text-meta">
          <Data className="[overflow-wrap:anywhere]">{session.path}</Data>
          {session.reasons.length > 0 && (
            <span>{`Reasons from ${server.name}: ${session.reasons.join(", ")}`}</span>
          )}
        </div>
      </Details>
    </div>
  );
}

/** One play: the file, who plays it on which player, and what the server does with it. */
export function SessionRow({ server, session }: { server: Server; session: Session }) {
  return (
    <ListRow
      title={fileName(session.path)}
      meta={
        <>
          {`${session.user} on ${session.player} · ${session.product} · Started `}
          <PlainTime at={session.startedAt} form="age" />
        </>
      }
      trailing={<Mark mark={decisionTerm(session)} />}
      detail={<SessionDetail server={server} session={session} />}
    />
  );
}

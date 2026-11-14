import type {
  MediaserversServerView as Server,
  MediaserversSessionView as Session,
} from "../../api/gen/model";
import { agoWords } from "../../format";
import { Chip } from "../../ui/chip";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { ListRow } from "../../ui/list";
import { fileName } from "./paths";
import { decisionLabels, decisionWords } from "./transcode-reasons";

function SessionDetail({ server, session }: { server: Server; session: Session }) {
  return (
    <div className="flex flex-col gap-1 pt-1 pb-1 text-meta text-ink-2">
      <p className="text-pretty">{decisionWords(server.name, session)}</p>
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
export function SessionRow({
  server,
  session,
  now,
}: {
  server: Server;
  session: Session;
  now: Date;
}) {
  return (
    <ListRow
      title={fileName(session.path)}
      meta={`${session.user} on ${session.player} · ${session.product} · Started ${agoWords(session.startedAt, now)}`}
      trailing={
        <Chip>
          {session.decision === "transcode" && session.hardware
            ? "Hardware transcode"
            : decisionLabels[session.decision]}
        </Chip>
      }
      detail={<SessionDetail server={server} session={session} />}
    />
  );
}

import type { MediaserversPlayView as Play } from "../../api/gen/model";
import { ListRow } from "../../ui/list";
import { PlainTime } from "../../ui/relative-time";
import { Mark } from "../../ui/state-mark";
import { fileName } from "./paths";
import { decisionTerm } from "./playback-words";

/** One play in a player's plays: the Work or file, who played it, when, and what the server did. */
export function PlayRow({ play }: { play: Play }) {
  const parts = [play.user, play.work === undefined ? "" : fileName(play.path)].filter(Boolean);
  return (
    <ListRow
      title={play.work?.title ?? fileName(play.path)}
      meta={
        <>
          {parts.map((part) => `${part} · `).join("")}
          <PlainTime at={play.startedAt} form="clock" />
          {play.endedAt === null ? " · Playing now" : ""}
        </>
      }
      trailing={<Mark mark={decisionTerm(play)} />}
    />
  );
}

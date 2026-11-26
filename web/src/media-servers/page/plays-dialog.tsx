import { useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { Suspense } from "react";

import type { MediaserversPlayView as Play } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { List, ListRow } from "../../ui/list";
import { PlainTime } from "../../ui/relative-time";
import { Skeleton } from "../../ui/skeleton";
import { Mark } from "../../ui/state-mark";
import { fileName } from "./paths";
// oxlint-disable-next-line import/max-dependencies -- a play row reads its file, time and decision mark, and the list pages
import { decisionTerm } from "./playback-words";
import { playsQuery } from "./plays-query";

function PlayRow({ play }: { play: Play }) {
  return (
    <ListRow
      title={play.work?.title ?? fileName(play.path)}
      meta={
        <>
          {`${play.user} · ${play.work === undefined ? "" : `${fileName(play.path)} · `}`}
          <PlainTime at={play.startedAt} form="clock" />
          {play.endedAt === null ? " · Playing now" : ""}
        </>
      }
      trailing={<Mark mark={decisionTerm(play)} />}
    />
  );
}

function Plays({ serverId, player, name }: { serverId: string; player: string; name: string }) {
  const { data, hasNextPage, fetchNextPage, isFetchingNextPage } = useSuspenseInfiniteQuery(
    playsQuery(serverId, player),
  );
  const plays = data.pages.flatMap((page) => page.plays);
  return (
    <div className="flex flex-col gap-4">
      <List label={`Plays on ${name}`}>
        {plays.map((play) => (
          <PlayRow key={play.id} play={play} />
        ))}
      </List>
      {hasNextPage && (
        <div>
          <Button
            busy={isFetchingNextPage}
            onClick={() => {
              void fetchNextPage();
            }}
          >
            Show older plays
          </Button>
        </div>
      )}
    </div>
  );
}

/** A player's plays on the server, newest first, opened from its Details. */
export function PlaysDialog({
  serverId,
  player,
  name,
}: {
  serverId: string;
  player: string;
  name: string;
}) {
  return (
    <Dialog title={`Plays on ${name}`} size={680} trigger={<Button>Show plays</Button>}>
      <Suspense
        fallback={<Skeleton label={`Loading the plays on ${name}`} shape="rows" lines={2} />}
      >
        <Plays serverId={serverId} player={player} name={name} />
      </Suspense>
    </Dialog>
  );
}

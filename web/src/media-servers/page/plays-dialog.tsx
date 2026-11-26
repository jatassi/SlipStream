import { useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { Suspense } from "react";

import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { List } from "../../ui/list";
import { Skeleton } from "../../ui/skeleton";
import { PlayRow } from "./play-row";
import { playsQuery } from "./plays-query";

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

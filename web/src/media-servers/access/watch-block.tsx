import { type ReactNode, useId } from "react";

import type {
  MediaserversMyAccessRowState as State,
  MediaserversMyServerAccess as Access,
} from "../../api/gen/model";
import { HowToWatch } from "./how-to-watch";
import { type BlockProps, PlexOffered, PlexShared, PlexWaiting } from "./plex-block";
import { UsersMaking, UsersOffered } from "./users-block";

type Block = (props: BlockProps) => ReactNode;

function Watching({ row, mine }: BlockProps) {
  return <HowToWatch row={row} plexPassLine={mine.plexPassLine} />;
}

const neverOn =
  (access: Access): Block =>
  ({ row }) => {
    throw new Error(`a ${access} row is never ${row.state}`);
  };

const users: { readonly [S in State]: Block } = {
  offered: UsersOffered,
  waiting: neverOn("users"),
  shared: neverOn("users"),
  making: UsersMaking,
  active: Watching,
};

const blocks: { readonly [A in Access]: { readonly [S in State]: Block } } = {
  plex_tv: {
    offered: PlexOffered,
    waiting: PlexWaiting,
    shared: PlexShared,
    making: neverOn("plex_tv"),
    active: Watching,
  },
  users,
};

export function WatchBlock(props: BlockProps) {
  const { row } = props;
  const Body = blocks[row.server.access][row.state];
  const titleId = useId();
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-4">
      <h2 id={titleId} className="text-heading text-ink-1">
        {row.server.name}
      </h2>
      <Body {...props} />
    </section>
  );
}

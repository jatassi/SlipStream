import { Tv01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type {
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
} from "../api/gen/model";
import { Icon } from "../ui/icon";
import { ListRow } from "../ui/list";
import { SignalLine } from "./lines";
import { kindOf } from "./server-form";

/** A server's row: its kind, name and reaching Node, then its state, which leads with what to do. */
export function ServerRow({ server, kinds }: { server: Server; kinds: readonly Kind[] }) {
  return (
    <ListRow
      title={server.name}
      meta={`${kindOf(kinds, server.kind).label} · Reached from ${server.reachingNode.name}`}
      leading={<Icon icon={Tv01Icon} className="size-5 shrink-0 text-ink-2" />}
      detail={
        <div className="pt-1 pb-1 pl-8">
          <SignalLine level={server.state.level} text={server.state.words} />
        </div>
      }
      render={<Link to="/settings/media-servers/$serverId" params={{ serverId: server.id }} />}
    />
  );
}

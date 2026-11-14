import { Link } from "@tanstack/react-router";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import { Chevron } from "../../ui/chevron";
import { Count } from "../../ui/count";
import { Icon } from "../../ui/icon";
import { List, ListRow } from "../../ui/list";
import { type ServerSection, serverSections } from "./sections";

export function SectionCount({ section, server }: { section: ServerSection; server: Server }) {
  const count = section.count?.(server) ?? 0;
  const dot = section.dot?.(server);
  return count > 0 || dot !== undefined ? <Count value={count} {...(dot && { dot })} /> : null;
}

/** On phone, the server's top page lists its other sections, each a pushed page. */
export function SectionRows({ server }: { server: Server }) {
  return (
    <div className="mb-7">
      <List label={server.name}>
        {serverSections
          .filter((section) => section.id !== "details")
          .map((section) => (
            <ListRow
              key={section.id}
              title={section.label}
              leading={<Icon icon={section.icon} className="size-5 shrink-0 text-ink-2" />}
              trailing={
                <span className="flex items-center gap-2">
                  <SectionCount section={section} server={server} />
                  <Chevron />
                </span>
              }
              render={<Link to={section.to} params={{ serverId: server.id }} />}
            />
          ))}
      </List>
    </div>
  );
}

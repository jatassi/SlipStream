import { ArrowUpRight01Icon } from "@hugeicons/core-free-icons";

import type { WorksOpenIn } from "../../api/gen/model";
import { Icon } from "../../ui/icon";

/** "Open in Plex", or the server's name when the Work is on two servers of one kind. */
function labelOf(link: WorksOpenIn, links: readonly WorksOpenIn[]): string {
  const shared = links.filter(({ kind }) => kind === link.kind).length > 1;
  return `Open in ${shared ? link.name : link.label}`;
}

/** A link to the Work in each Media Server's web app that holds it, in a new tab; nothing when none does. */
export function OpenIn({ links }: { links: readonly WorksOpenIn[] }) {
  if (links.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-label text-ink-3">Media Servers</h3>
      <ul aria-label="Open in" className="flex flex-wrap gap-2">
        {links.map((link) => (
          <li key={link.serverId}>
            <a
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              title={link.name}
              className="inline-flex h-(--control-height) items-center gap-2 rounded-capsule bg-fill px-3.5 text-button font-semibold text-ink-1 ring-1 ring-hairline-strong ring-inset hover:bg-fill-hover"
            >
              <Icon icon={ArrowUpRight01Icon} className="size-4" />
              {labelOf(link, links)}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

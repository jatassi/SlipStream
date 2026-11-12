import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { folderEntries, hdTarget, packEntry, takenEntry } from "./import-entries";
import { type ImportsWorld, changedOver, serveImports } from "./imports";
import { cinema } from "./library";
import { liveServer } from "./live-server";
import { home, nodesAre, seedbox } from "./nodes";
import { shows } from "./pending-imports";
import { renderApp } from "./render-app";
import { keeper, starter, targetHandlers } from "./targets";

/** Pending imports over a world whose changes reach the page live, with Films and Shows. */
export function openPending({
  start = {},
  path = "/library/pending",
  handlers = [],
}: { start?: Partial<ImportsWorld> } & Parameters<typeof renderApp>[0] = {}) {
  const live = liveServer();
  const served = serveImports({
    changed: changedOver(live),
    imports: [packEntry, takenEntry],
    folder: folderEntries,
    ...start,
  });
  const app = renderApp({
    path,
    handlers: [
      ...live.handlers,
      getListContentTypesMockHandler({ contentTypes: [cinema, shows] }),
      ...targetHandlers([starter, keeper, hdTarget]),
      nodesAre(home, seedbox),
      ...handlers,
    ],
  });
  return { ...served, app };
}

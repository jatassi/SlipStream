import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { type ImportsWorld, changedOver, serveImports } from "./imports";
import { cinema } from "./library";
import { liveServer } from "./live-server";
import { shows } from "./pending-imports";
import { renderApp } from "./render-app";
import { targetHandlers } from "./targets";

/** Pending imports over a world whose changes reach the page live, with Films and Shows. */
export function openPending({
  start = {},
  path = "/library/pending",
  handlers = [],
}: { start?: Partial<ImportsWorld> } & Parameters<typeof renderApp>[0] = {}) {
  const live = liveServer();
  const served = serveImports({ changed: changedOver(live), ...start });
  const app = renderApp({
    path,
    handlers: [
      ...live.handlers,
      getListContentTypesMockHandler({ contentTypes: [cinema, shows] }),
      ...targetHandlers(),
      ...handlers,
    ],
  });
  return { ...served, app };
}

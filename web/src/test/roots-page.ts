import { page, userEvent } from "vitest/browser";

import type { RootsRoot, WorksContentType } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { type FakeNode, serveFolders, unixFolder } from "./folders";
import { cinema } from "./library";
import { nodesAre } from "./nodes";
import { renderApp } from "./render-app";
import { media, notStorage } from "./roots";

export const zines: WorksContentType = {
  ...cinema,
  id: "zine",
  name: "Zines",
  work: { one: "Zine", many: "Zines" },
  item: { one: "Issue", many: "Issues" },
};

const mediaFolders: FakeNode = {
  places: [],
  folders: {
    "/srv": unixFolder("/srv", ["media"]),
    "/srv/media": unixFolder("/srv/media", ["films", "zines"]),
    "/srv/media/zines": unixFolder("/srv/media/zines"),
    "/srv/media/films": unixFolder("/srv/media/films"),
  },
};

/** Settings > Library roots with Films and Zines, on a Controller holding Storage and a Worker that doesn't. */
export function openRoots(more: Parameters<typeof renderApp>[0] = {}) {
  serveFolders({ [media.id]: mediaFolders });
  return renderApp({
    path: "/settings/library-roots",
    handlers: [
      ...(more.handlers ?? []),
      nodesAre(media, notStorage),
      getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
    ],
  });
}

export async function act(root: RootsRoot, action: string) {
  await userEvent.click(page.getByRole("button", { name: `Actions for ${root.path}` }));
  await userEvent.click(page.getByRole("menuitem", { name: action }));
}

/** A request held until the test lets it answer. */
export function gate() {
  let open!: () => void;
  const held = new Promise<void>((resolve) => {
    open = resolve;
  });
  return { held, open };
}

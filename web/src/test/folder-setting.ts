import { HttpResponse, http } from "msw";

import type { CheckFolderInputBody, RootsVerdict } from "../api/gen/model";
import { getListRootsMockHandler } from "../api/gen/query/roots/roots.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { CheckFolderBody } from "../api/gen/zod/roots/roots.zod";
import { type FakeNode, serveFolders, unixFolder } from "./folders";
import { cinema } from "./library";
import { nodesAre } from "./nodes";
import { filmsRoot, media, mediaVolume } from "./roots";
import { worker } from "./worker";

const mediaFolders: FakeNode = {
  places: [],
  folders: {
    "/srv": unixFolder("/srv", ["media", "torrents"]),
    "/srv/torrents": unixFolder("/srv/torrents"),
    "/srv/media": unixFolder("/srv/media", ["films", "incoming"]),
    "/srv/media/incoming": unixFolder("/srv/media/incoming"),
  },
};

/**
 * Serves media's folders, its Films root and a folder check that refuses
 * any folder in that root, and answers each check asked.
 */
export function serveFolderSetting(): CheckFolderInputBody[] {
  const asked: CheckFolderInputBody[] = [];
  serveFolders({ [media.id]: mediaFolders });
  worker.use(
    nodesAre(media),
    getListRootsMockHandler({ roots: [filmsRoot] }),
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    http.post("*/api/v1/nodes/:id/folders/check", async ({ request }) => {
      const body = CheckFolderBody.parse(await request.json());
      asked.push(body);
      const verdict: RootsVerdict = body.path.startsWith(filmsRoot.path)
        ? {
            usable: false,
            path: body.path,
            refusal: { reason: "inside_root", rootId: filmsRoot.id },
          }
        : { usable: true, path: body.path, volume: mediaVolume };
      return HttpResponse.json(verdict);
    }),
  );
  return asked;
}

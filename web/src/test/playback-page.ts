import type { RequestHandler } from "msw";

import type { PlaybackReport } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { cinema } from "./library";
import { mediaServersWorld } from "./media-server-writes";
import { denJellyfin, livingRoomPlex } from "./media-servers";
import { denReport, playbackHandlers, plexReport } from "./playback";
import { renderApp } from "./render-app";

/** One server's Playback report, with Den Jellyfin's and Living room Plex's own reports unless one is given. */
export function openPlayback(
  server = denJellyfin,
  shown?: PlaybackReport,
  handlers: RequestHandler[] = [],
) {
  renderApp({
    path: `/settings/media-servers/${server.id}/playback`,
    handlers: [
      ...handlers,
      ...playbackHandlers({
        [denJellyfin.id]: denReport,
        [livingRoomPlex.id]: plexReport,
        ...(shown && { [server.id]: shown }),
      }),
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      ...mediaServersWorld([denJellyfin, livingRoomPlex]).handlers,
    ],
  });
}

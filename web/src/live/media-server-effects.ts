import {
  getGetMediaServerQueryKey,
  getListMediaServerChangesQueryKey,
  getListMediaServerCredentialHoldersQueryKey,
  getListMediaServerLibrariesQueryKey,
  getListMediaServerSessionsQueryKey,
  getListMediaServersQueryKey,
} from "../api/gen/query/mediaservers/mediaservers";
import { getGetWorkQueryKey } from "../api/gen/query/works/works";
import type { LiveEffect, LiveEffects, PayloadOf } from "./types";

const health = (
  { serverId }: PayloadOf<"media-servers.unhealthy" | "media-servers.recovered">,
  { refresh }: Parameters<LiveEffect<"media-servers.unhealthy">>[1],
) => {
  refresh({ queryKey: getListMediaServersQueryKey() });
  refresh({ queryKey: getGetMediaServerQueryKey(serverId) });
};

/** Media Servers' records, queues and sessions, and the Open in links a lookup fills in. */
export const mediaServerEffects = {
  "media-servers.changed": ({ serverIds }, { refresh }) => {
    refresh({ queryKey: getListMediaServersQueryKey() });
    for (const id of serverIds) {
      refresh({ queryKey: getGetMediaServerQueryKey(id) });
      refresh({ queryKey: getListMediaServerChangesQueryKey(id) });
      refresh({ queryKey: getListMediaServerLibrariesQueryKey(id) });
      refresh({ queryKey: getListMediaServerCredentialHoldersQueryKey(id) });
    }
  },
  "media-servers.sessions-changed": ({ serverIds }, { refresh }) => {
    for (const id of serverIds) {
      refresh({ queryKey: getListMediaServerSessionsQueryKey(id) });
    }
  },
  "media-servers.unhealthy": health,
  "media-servers.recovered": health,
  "media-servers.items-found": ({ workIds }, { refresh }) => {
    for (const id of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(id) });
    }
  },
} satisfies Pick<
  LiveEffects,
  | "media-servers.changed"
  | "media-servers.sessions-changed"
  | "media-servers.unhealthy"
  | "media-servers.recovered"
  | "media-servers.items-found"
>;

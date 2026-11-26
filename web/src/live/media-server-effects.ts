import {
  getGetMediaServerPlaybackQueryKey,
  getGetMediaServerQueryKey,
  getListMediaServerChangesQueryKey,
  getListMediaServerCredentialHoldersQueryKey,
  getListMediaServerLibrariesQueryKey,
  getListMediaServerPlaysQueryKey,
  getListMediaServerSessionsQueryKey,
  getListMediaServersQueryKey,
  getListMediaServerAccessQueryKey,
  getListMyMediaServerAccessQueryKey,
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

/** Media Servers' records, queues, sessions and plays, Members' access, and the Open in links a lookup fills in. */
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
  "media-servers.plays-changed": ({ serverIds }, { refresh }) => {
    for (const id of serverIds) {
      refresh({ queryKey: getGetMediaServerPlaybackQueryKey(id) });
      refresh({ queryKey: getListMediaServerPlaysQueryKey(id) });
    }
  },
  "media-servers.unhealthy": health,
  "media-servers.recovered": health,
  "media-servers.items-found": ({ workIds }, { refresh }) => {
    for (const id of workIds) {
      refresh({ queryKey: getGetWorkQueryKey(id) });
    }
  },
  "media-servers.access-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListMediaServerAccessQueryKey() });
    refresh({ queryKey: getListMediaServersQueryKey() });
  },
  "media-servers.my-access-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListMyMediaServerAccessQueryKey() });
  },
} satisfies Pick<
  LiveEffects,
  | "media-servers.changed"
  | "media-servers.sessions-changed"
  | "media-servers.plays-changed"
  | "media-servers.unhealthy"
  | "media-servers.recovered"
  | "media-servers.items-found"
  | "media-servers.access-changed"
  | "media-servers.my-access-changed"
>;

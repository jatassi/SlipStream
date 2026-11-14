import type { QueryClient } from "@tanstack/react-query";

import type {
  MediaserversLibraries,
  MediaserversMappingView,
  MediaserversServerView,
  MediaserversSettings,
  ServersOutputBody,
} from "../api/gen/model";
import {
  getGetMediaServerQueryKey,
  getGetMediaServerSettingsQueryKey,
  getGetPlexAccountQueryKey,
  getListMediaServerLibrariesQueryKey,
  getListMediaServersQueryKey,
} from "../api/gen/query/mediaservers/mediaservers";
import { getGetTaskQueryKey, getListTasksQueryKey } from "../api/gen/query/tasks/tasks";

// A save answers with the server or its mappings as written, so the page shows
// them at once rather than after media-servers.changed refetches them.

export function adoptMediaServer(queryClient: QueryClient, saved: MediaserversServerView) {
  queryClient.setQueryData(getGetMediaServerQueryKey(saved.id), saved);
  queryClient.setQueryData<ServersOutputBody>(
    getListMediaServersQueryKey(),
    (list) =>
      list && {
        ...list,
        servers: list.servers.some((shown) => shown.id === saved.id)
          ? list.servers.map((shown) => (shown.id === saved.id ? saved : shown))
          : [...list.servers, saved],
      },
  );
}

export function adoptMappings(
  queryClient: QueryClient,
  serverId: string,
  mappings: MediaserversMappingView[],
) {
  queryClient.setQueryData<MediaserversLibraries>(
    getListMediaServerLibrariesQueryKey(serverId),
    (libraries) => libraries && { ...libraries, mappings },
  );
}

export function dropMediaServer(queryClient: QueryClient, id: string) {
  queryClient.setQueryData<ServersOutputBody>(
    getListMediaServersQueryKey(),
    (list) => list && { ...list, servers: list.servers.filter((shown) => shown.id !== id) },
  );
  queryClient.removeQueries({ queryKey: getGetMediaServerQueryKey(id) });
}

const renewalTask = "plex-sign-in";

// The Plex sign-in renewal's schedule and margin show on the Tasks page too,
// which reads them from the task and hears no event when they change here.
const renewalKeys = new Set(["tasks.plex-sign-in.cadence", "mediaServers.renewMargin"]);

export async function adoptMediaServerSettings(
  queryClient: QueryClient,
  key: string,
  saved: MediaserversSettings,
) {
  queryClient.setQueryData(getGetMediaServerSettingsQueryKey(), saved);
  if (renewalKeys.has(key)) {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: getListTasksQueryKey() }),
      queryClient.invalidateQueries({ queryKey: getGetTaskQueryKey(renewalTask) }),
    ]);
  }
}

/** Signing out of Plex removes every Plex server, as the Controller did. */
export async function adoptPlexSignOut(queryClient: QueryClient, removed: readonly string[]) {
  for (const id of removed) {
    dropMediaServer(queryClient, id);
  }
  await queryClient.invalidateQueries({ queryKey: getGetPlexAccountQueryKey() });
}

/** A Plex sign-in that signed in changes the account. It resolves once the account is read again. */
export async function adoptPlexSignIn(queryClient: QueryClient) {
  await queryClient.invalidateQueries({ queryKey: getGetPlexAccountQueryKey() });
}

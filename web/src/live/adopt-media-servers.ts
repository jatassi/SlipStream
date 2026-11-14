import type { QueryClient } from "@tanstack/react-query";

import type {
  MediaserversLibraries,
  MediaserversMappingView,
  MediaserversServerView,
  ServersOutputBody,
} from "../api/gen/model";
import {
  getGetMediaServerQueryKey,
  getListMediaServerLibrariesQueryKey,
  getListMediaServersQueryKey,
} from "../api/gen/query/mediaservers/mediaservers";

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

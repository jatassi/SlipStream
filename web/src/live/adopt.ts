import type { QueryClient } from "@tanstack/react-query";

import type {
  DownloadclientsClientsList,
  DownloadclientsSettings,
  DownloadclientsThresholds,
  SystemNodeConfig,
  TasksOverview,
  TasksSchedule,
} from "../api/gen/model";
import {
  getGetDownloadClientSettingsQueryKey,
  getListDownloadClientsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
import { getGetNodeConfigQueryKey } from "../api/gen/query/system/system";
import { getListTasksQueryKey } from "../api/gen/query/tasks/tasks";

// A save answers with the settings as written, so the page shows them at once
// rather than after the live event's refetch.

export function adoptNodeConfig(
  queryClient: QueryClient,
  nodeId: string,
  config: SystemNodeConfig,
) {
  queryClient.setQueryData(getGetNodeConfigQueryKey(nodeId), config);
}

export function adoptTaskSettings(queryClient: QueryClient, settings: TasksSchedule) {
  queryClient.setQueryData<TasksOverview>(
    getListTasksQueryKey(),
    (overview) => overview && { ...overview, settings },
  );
}

export function adoptClientSettings(
  queryClient: QueryClient,
  nodeId: string,
  settings: DownloadclientsSettings,
) {
  queryClient.setQueryData(getGetDownloadClientSettingsQueryKey(nodeId), settings);
}

export function adoptThresholds(queryClient: QueryClient, thresholds: DownloadclientsThresholds) {
  queryClient.setQueryData<DownloadclientsClientsList>(
    getListDownloadClientsQueryKey(),
    (list) => list && { ...list, thresholds },
  );
}

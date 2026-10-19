import type { QueryClient } from "@tanstack/react-query";

import type {
  HealthReport,
  HealthSettings,
  HealthVolume,
  HealthVolumeList,
  RootsRootSettings,
  SystemNodeConfig,
  TasksOverview,
  TasksSchedule,
} from "../api/gen/model";
import {
  getGetHealthSettingsQueryKey,
  getGetSystemHealthQueryKey,
  getListHealthVolumesQueryKey,
} from "../api/gen/query/health/health";
import { getGetRootSettingsQueryKey } from "../api/gen/query/roots/roots";
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

export function adoptHealthReport(queryClient: QueryClient, report: HealthReport) {
  queryClient.setQueryData(getGetSystemHealthQueryKey(), report);
}

export function adoptHealthSettings(queryClient: QueryClient, settings: HealthSettings) {
  queryClient.setQueryData(getGetHealthSettingsQueryKey(), settings);
}

export function adoptRootSettings(queryClient: QueryClient, settings: RootsRootSettings) {
  queryClient.setQueryData(getGetRootSettingsQueryKey(), settings);
}

const sameVolume = (a: { nodeId: string; mount: string }, b: { nodeId: string; mount: string }) =>
  a.nodeId === b.nodeId && a.mount === b.mount;

export function adoptVolumeLevels(queryClient: QueryClient, volume: HealthVolume) {
  queryClient.setQueryData<HealthVolumeList>(
    getListHealthVolumesQueryKey(),
    (list) =>
      list && {
        ...list,
        volumes: list.volumes.map((shown) => (sameVolume(shown, volume) ? volume : shown)),
      },
  );
}

export function dropVolumeLevels(
  queryClient: QueryClient,
  removed: { nodeId: string; mount: string },
) {
  queryClient.setQueryData<HealthVolumeList>(
    getListHealthVolumesQueryKey(),
    (list) =>
      list && {
        ...list,
        notReporting: list.notReporting.filter((shown) => !sameVolume(shown, removed)),
      },
  );
}

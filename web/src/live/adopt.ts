import type { QueryClient } from "@tanstack/react-query";

import type {
  DownloadclientsClientsList,
  DownloadclientsExternalClient,
  DownloadclientsSettings,
  DownloadclientsThresholds,
  HealthReport,
  HealthSettings,
  HealthVolume,
  HealthVolumeList,
  IndexersChallengeSolver,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersIndexerList,
  IndexersSettings,
  IndexersTracker,
  IndexersTrackerList,
  RootsRootSettings,
  SystemNodeConfig,
  TasksOverview,
  TasksSchedule,
  TransfersSettings,
  WorksContentType,
  WorksContentTypeList,
} from "../api/gen/model";
import {
  getGetDownloadClientSettingsQueryKey,
  getListDownloadClientsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
import {
  getGetHealthSettingsQueryKey,
  getGetSystemHealthQueryKey,
  getListHealthVolumesQueryKey,
} from "../api/gen/query/health/health";
import {
  getGetChallengeSolverQueryKey,
  getGetIndexerCatalogQueryKey,
  getGetIndexerQueryKey,
  getGetIndexerSettingsQueryKey,
  getGetTrackerQueryKey,
  getListIndexersQueryKey,
  getListTrackersQueryKey,
} from "../api/gen/query/indexers/indexers";
import { getGetRootSettingsQueryKey } from "../api/gen/query/roots/roots";
import { getGetNodeConfigQueryKey } from "../api/gen/query/system/system";
import { getListTasksQueryKey } from "../api/gen/query/tasks/tasks";
import { getGetTransferSettingsQueryKey } from "../api/gen/query/transfers/transfers";
import { getListContentTypesQueryKey } from "../api/gen/query/works/works";

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

export function adoptChallengeSolver(
  queryClient: QueryClient,
  nodeId: string,
  solver: IndexersChallengeSolver,
) {
  queryClient.setQueryData(getGetChallengeSolverQueryKey(nodeId), solver);
}

export function adoptTransferSettings(
  queryClient: QueryClient,
  nodeId: string,
  settings: TransfersSettings,
) {
  queryClient.setQueryData(getGetTransferSettingsQueryKey(nodeId), settings);
}

export function adoptThresholds(queryClient: QueryClient, thresholds: DownloadclientsThresholds) {
  queryClient.setQueryData<DownloadclientsClientsList>(
    getListDownloadClientsQueryKey(),
    (list) => list && { ...list, thresholds },
  );
}

const byName = (a: DownloadclientsExternalClient, b: DownloadclientsExternalClient) =>
  a.name.localeCompare(b.name);

/** A saved external client on its Node's card, taken off any card it moved from. */
export function adoptExternalClient(
  queryClient: QueryClient,
  saved: DownloadclientsExternalClient,
) {
  queryClient.setQueryData<DownloadclientsClientsList>(
    getListDownloadClientsQueryKey(),
    (list) =>
      list && {
        ...list,
        cards: list.cards.map((card) => {
          const others = card.externals.filter((shown) => shown.id !== saved.id);
          const externals =
            card.nodeId === saved.nodeId ? [...others, saved].toSorted(byName) : others;
          return { ...card, externals };
        }),
        orphans: list.orphans.map((shown) => (shown.id === saved.id ? saved : shown)),
      },
  );
}

export function dropExternalClient(queryClient: QueryClient, id: string) {
  queryClient.setQueryData<DownloadclientsClientsList>(
    getListDownloadClientsQueryKey(),
    (list) =>
      list && {
        ...list,
        cards: list.cards.map((card) => ({
          ...card,
          externals: card.externals.filter((shown) => shown.id !== id),
        })),
        orphans: list.orphans.filter((shown) => shown.id !== id),
      },
  );
}

export function adoptGrabsTo(
  queryClient: QueryClient,
  nodeId: string,
  settings: DownloadclientsSettings,
) {
  adoptClientSettings(queryClient, nodeId, settings);
  queryClient.setQueryData<DownloadclientsClientsList>(
    getListDownloadClientsQueryKey(),
    (list) =>
      list && {
        ...list,
        cards: list.cards.map((card) =>
          card.nodeId === nodeId ? { ...card, grabsTo: settings.grabsTo } : card,
        ),
      },
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

export function adoptContentType(queryClient: QueryClient, saved: WorksContentType) {
  queryClient.setQueryData(
    getListContentTypesQueryKey(),
    (list: WorksContentTypeList | undefined) =>
      list && {
        contentTypes: list.contentTypes.map((type) => (type.id === saved.id ? saved : type)),
      },
  );
}

export function adoptIndexer(queryClient: QueryClient, saved: IndexersIndexer) {
  queryClient.setQueryData(getGetIndexerQueryKey(saved.id), saved);
  queryClient.setQueryData<IndexersIndexerList>(
    getListIndexersQueryKey(),
    (list) =>
      list && { indexers: list.indexers.map((shown) => (shown.id === saved.id ? saved : shown)) },
  );
}

export function dropIndexer(queryClient: QueryClient, id: string) {
  queryClient.setQueryData<IndexersIndexerList>(
    getListIndexersQueryKey(),
    (list) => list && { indexers: list.indexers.filter((shown) => shown.id !== id) },
  );
}

export function adoptCatalog(queryClient: QueryClient, catalog: IndexersIndexerCatalog) {
  queryClient.setQueryData(getGetIndexerCatalogQueryKey(), catalog);
}

export function adoptIndexerSettings(queryClient: QueryClient, settings: IndexersSettings) {
  queryClient.setQueryData(getGetIndexerSettingsQueryKey(), settings);
}

export function adoptTracker(queryClient: QueryClient, saved: IndexersTracker) {
  queryClient.setQueryData(getGetTrackerQueryKey(saved.id), saved);
  queryClient.setQueryData<IndexersTrackerList>(
    getListTrackersQueryKey(),
    (list) =>
      list && { trackers: list.trackers.map((shown) => (shown.id === saved.id ? saved : shown)) },
  );
}

/** from is gone, and kept now lists from's Indexers. */
export function adoptMerge(queryClient: QueryClient, from: string, kept: IndexersTracker) {
  queryClient.removeQueries({ queryKey: getGetTrackerQueryKey(from), exact: true });
  queryClient.setQueryData(getGetTrackerQueryKey(kept.id), kept);
  queryClient.setQueryData<IndexersTrackerList>(
    getListTrackersQueryKey(),
    (list) =>
      list && {
        trackers: list.trackers
          .filter((shown) => shown.id !== from)
          .map((shown) => (shown.id === kept.id ? kept : shown)),
      },
  );
}

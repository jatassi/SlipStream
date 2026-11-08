import type { QueryClient } from "@tanstack/react-query";

import type {
  TrackersGoal,
  TrackersSettings,
  TrackersTracker,
  TrackersTrackerItem,
  TrackersTrackerList,
  TrackersTrackerPage,
} from "../api/gen/model";
import {
  getGetDefaultGoalQueryKey,
  getGetSeedingSettingsQueryKey,
  getGetTrackerQueryKey,
  getListTrackersQueryKey,
} from "../api/gen/query/trackers/trackers";

// A save answers the Tracker alone, so its page keeps its stats and its
// list item its counts.
function adopted(shown: TrackersTrackerItem, saved: TrackersTracker): TrackersTrackerItem {
  return shown.id === saved.id ? { ...shown, ...saved } : shown;
}

export function adoptTracker(queryClient: QueryClient, saved: TrackersTracker) {
  queryClient.setQueryData<TrackersTrackerPage>(
    getGetTrackerQueryKey(saved.id),
    (page) => page && { ...page, ...saved },
  );
  queryClient.setQueryData<TrackersTrackerList>(
    getListTrackersQueryKey(),
    (list) => list && { trackers: list.trackers.map((shown) => adopted(shown, saved)) },
  );
}

/** from is gone, and kept now lists from's Indexers. */
export function adoptMerge(queryClient: QueryClient, from: string, kept: TrackersTracker) {
  queryClient.removeQueries({ queryKey: getGetTrackerQueryKey(from), exact: true });
  queryClient.setQueryData<TrackersTrackerPage>(
    getGetTrackerQueryKey(kept.id),
    (page) => page && { ...page, ...kept },
  );
  queryClient.setQueryData<TrackersTrackerList>(
    getListTrackersQueryKey(),
    (list) =>
      list && {
        trackers: list.trackers
          .filter((shown) => shown.id !== from)
          .map((shown) => adopted(shown, kept)),
      },
  );
}

/** id is removed: the list drops it at once, and its page's data once the page has closed. */
export function adoptRemoved(queryClient: QueryClient, id: string) {
  queryClient.setQueryData<TrackersTrackerList>(
    getListTrackersQueryKey(),
    (list) => list && { trackers: list.trackers.filter((shown) => shown.id !== id) },
  );
  return () => {
    queryClient.removeQueries({ queryKey: getGetTrackerQueryKey(id), exact: true });
  };
}

export function adoptDefaultGoal(queryClient: QueryClient, goal: TrackersGoal) {
  queryClient.setQueryData(getGetDefaultGoalQueryKey(), goal);
}

export function adoptSeedingSettings(queryClient: QueryClient, settings: TrackersSettings) {
  queryClient.setQueryData(getGetSeedingSettingsQueryKey(), settings);
}

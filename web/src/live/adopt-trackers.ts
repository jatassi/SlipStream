import type { QueryClient } from "@tanstack/react-query";

import type { TrackersTracker, TrackersTrackerList } from "../api/gen/model";
import { getGetTrackerQueryKey, getListTrackersQueryKey } from "../api/gen/query/trackers/trackers";

export function adoptTracker(queryClient: QueryClient, saved: TrackersTracker) {
  queryClient.setQueryData(getGetTrackerQueryKey(saved.id), saved);
  queryClient.setQueryData<TrackersTrackerList>(
    getListTrackersQueryKey(),
    (list) =>
      list && { trackers: list.trackers.map((shown) => (shown.id === saved.id ? saved : shown)) },
  );
}

/** from is gone, and kept now lists from's Indexers. */
export function adoptMerge(queryClient: QueryClient, from: string, kept: TrackersTracker) {
  queryClient.removeQueries({ queryKey: getGetTrackerQueryKey(from), exact: true });
  queryClient.setQueryData(getGetTrackerQueryKey(kept.id), kept);
  queryClient.setQueryData<TrackersTrackerList>(
    getListTrackersQueryKey(),
    (list) =>
      list && {
        trackers: list.trackers
          .filter((shown) => shown.id !== from)
          .map((shown) => (shown.id === kept.id ? kept : shown)),
      },
  );
}

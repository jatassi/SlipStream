import { notFound } from "@tanstack/react-router";

import type { TrackersTrackerItem } from "../api/gen/model";
import { useListTrackersSuspense } from "../api/gen/query/trackers/trackers";

export function trackerOf(
  trackers: readonly TrackersTrackerItem[],
  trackerId: string,
): TrackersTrackerItem {
  const tracker = trackers.find((candidate) => candidate.id === trackerId);
  if (tracker === undefined) {
    // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
    throw notFound();
  }
  return tracker;
}

/** A Tracker as the Trackers list shows it, with its counts; its stats are on its page. */
export const useTracker = (trackerId: string): TrackersTrackerItem =>
  trackerOf(useListTrackersSuspense().data.trackers, trackerId);

export const useTrackers = (): readonly TrackersTrackerItem[] =>
  useListTrackersSuspense().data.trackers;

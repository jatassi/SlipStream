import { Download04Icon, Search01Icon, Plant01Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { TrackersTrackerItem } from "../api/gen/model";
import type { AlertTone } from "../ui/count";

export type TrackerSection = {
  readonly id: string;
  readonly label: string;
  readonly icon: IconSvgElement;
  readonly to: `/settings/trackers/$trackerId${string}`;
  /** The number beside the section's tab or row, shown above zero. */
  readonly count: (tracker: TrackersTrackerItem) => number;
  readonly dot?: (tracker: TrackersTrackerItem) => AlertTone | undefined;
};

const atRisk = (tracker: TrackersTrackerItem): AlertTone | undefined =>
  tracker.counts.atRisk > 0 ? "warn" : undefined;

/** A private Tracker with no rule never ends its torrents' seeding. */
export const wantsRule = (tracker: TrackersTrackerItem): boolean =>
  (tracker.privacy === "private" || tracker.privacy === "semi-private") && tracker.rule === null;

/**
 * A Tracker page's sections, in order. They draw the desktop page tabs and
 * the phone page's rows. The first is the page's index.
 */
export const trackerSections = [
  {
    id: "torrents",
    label: "Torrents",
    icon: Download04Icon,
    to: "/settings/trackers/$trackerId",
    count: (tracker) => tracker.counts.torrents,
    dot: atRisk,
  },
  {
    id: "seeding",
    label: "Seeding",
    icon: Plant01Icon,
    to: "/settings/trackers/$trackerId/seeding",
    count: (tracker) => tracker.counts.owing,
    dot: (tracker) => (wantsRule(tracker) ? "warn" : atRisk(tracker)),
  },
  {
    id: "indexers",
    label: "Indexers",
    icon: Search01Icon,
    to: "/settings/trackers/$trackerId/indexers",
    count: (tracker) => tracker.indexerIds.length,
  },
] as const satisfies readonly TrackerSection[];

export type TrackerSectionId = (typeof trackerSections)[number]["id"];

export function trackerSectionOf(id: TrackerSectionId): TrackerSection {
  const section = trackerSections.find((candidate) => candidate.id === id);
  if (section === undefined) {
    throw new Error(`no Tracker section ${id}`);
  }
  return section;
}

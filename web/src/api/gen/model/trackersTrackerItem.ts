import type { TrackersCounts } from "./trackersCounts.ts";
import type { TrackersGoal } from "./trackersGoal.ts";
import type { TrackersRule } from "./trackersRule.ts";
import type { TrackersTrackerItemPrivacy } from "./trackersTrackerItemPrivacy.ts";

export interface TrackersTrackerItem {
  /** Its torrents, live or being removed, as Activity > Torrents counts them */
  counts: TrackersCounts;
  /** Its own seeding goal; null while it follows the default goal */
  goal: TrackersGoal | null;
  /** The announce hosts of its private torrents */
  hosts: string[];
  id: string;
  /** The Indexers that reach it, by name */
  indexerIds: string[];
  /** How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer> */
  keys: string[];
  name: string;
  privacy: TrackersTrackerItemPrivacy;
  /** Its seeding rule in force; null while none is known, and always for a public Tracker */
  rule: TrackersRule | null;
}

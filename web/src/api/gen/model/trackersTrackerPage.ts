import type { TrackersAttention } from "./trackersAttention.ts";
import type { TrackersGoal } from "./trackersGoal.ts";
import type { TrackersRule } from "./trackersRule.ts";
import type { TrackersStats } from "./trackersStats.ts";
import type { TrackersTrackerPagePrivacy } from "./trackersTrackerPagePrivacy.ts";

export interface TrackersTrackerPage {
  /** What needs a look on its page, worst first */
  attention: TrackersAttention[];
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
  privacy: TrackersTrackerPagePrivacy;
  /** Its seeding rule in force; null while none is known, and always for a public Tracker */
  rule: TrackersRule | null;
  /** SlipStream's own sums over its torrents */
  stats: TrackersStats;
}

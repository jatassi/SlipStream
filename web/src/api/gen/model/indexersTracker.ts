import type { IndexersTrackerPrivacy } from "./indexersTrackerPrivacy.ts";

export interface IndexersTracker {
  /** The announce hosts of its private torrents */
  hosts: string[];
  id: string;
  /** The Indexers that reach it, by name */
  indexerIds: string[];
  /** How a Release finds it: definition:<id>, prowlarr:<feed Indexer>:<source id> or named:<Indexer> */
  keys: string[];
  name: string;
  privacy: IndexersTrackerPrivacy;
}

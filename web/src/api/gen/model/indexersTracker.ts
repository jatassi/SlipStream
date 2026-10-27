import type { IndexersTrackerPrivacy } from "./indexersTrackerPrivacy.ts";

export interface IndexersTracker {
  id: string;
  /** The Indexers that reach it, by name */
  indexerIds: string[];
  name: string;
  privacy: IndexersTrackerPrivacy;
}

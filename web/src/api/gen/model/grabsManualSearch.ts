import type { GrabsFolded } from "./grabsFolded.ts";
import type { GrabsIndexerLine } from "./grabsIndexerLine.ts";
import type { GrabsManualScope } from "./grabsManualScope.ts";
import type { GrabsPinned } from "./grabsPinned.ts";
import type { GrabsPlan } from "./grabsPlan.ts";

export interface GrabsManualSearch {
  /** The Releases SlipStream wouldn't grab, in groups */
  folded: GrabsFolded;
  id: string;
  indexers: GrabsIndexerLine[];
  /** Such as Searching 6 Indexers. 4 have responded. */
  line: string;
  /** The pins still in effect; send these next */
  pins: GrabsPinned[];
  plan: GrabsPlan;
  /**
   * The typed query; null when each run sent its Item's terms
   * @nullable
   */
  query: string | null;
  /** The scope is one Item, so a typed query can search it */
  queryable: boolean;
  /**
   * How many Releases listManualSearchReleases lists
   * @minimum 0
   */
  releaseCount: number;
  /** What it was started for; no levels, Items or cell for the whole Work */
  scope: GrabsManualScope;
  /** An Indexer asked hasn't responded yet */
  searching: boolean;
  startedAt: string;
  /** A name, or the API key */
  startedBy: string;
  /** The scope in words */
  title: string;
}

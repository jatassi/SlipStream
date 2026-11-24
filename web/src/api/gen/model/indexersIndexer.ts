import type { IndexersIndexerKind } from "./indexersIndexerKind.ts";
import type { IndexersLimitUse } from "./indexersLimitUse.ts";
import type { IndexersOutcome } from "./indexersOutcome.ts";
import type { IndexersPin } from "./indexersPin.ts";
import type { IndexersPortPin } from "./indexersPortPin.ts";
import type { IndexersRssFeed } from "./indexersRssFeed.ts";
import type { IndexersSearchMode } from "./indexersSearchMode.ts";
import type { IndexersSession } from "./indexersSession.ts";
import type { IndexersStatus } from "./indexersStatus.ts";

export interface IndexersIndexer {
  /** Read only for an Indexer that signs in */
  allowFailover: boolean;
  /** An RSS feed whose items may have no size */
  allowNoSize: boolean;
  automaticSearch: boolean;
  /**
   * Newznab category ids; null for every category it advertises
   * @nullable
   */
  categories: number[] | null;
  createdAt: string;
  /** A catalog Indexer's definition; null for every other kind */
  definition: IndexersPin | null;
  /**
   * The host of a feed's address, which is never returned
   * @nullable
   */
  feedHost: string | null;
  /**
   * What the Admin can do about its status, in words; null when nothing
   * @nullable
   */
  fix: string | null;
  grabs: IndexersLimitUse;
  /** The Node it searches from */
  homeNodeId: string;
  id: string;
  kind: IndexersIndexerKind;
  /**
   * The Content Types an RSS feed carries; null for every other kind
   * @nullable
   */
  media: string[] | null;
  name: string;
  /** What changed when SlipStream last moved it to a new definition version or from a built-in site to its catalog definition, for the Indexer page */
  notes: string[];
  on: boolean;
  /** The built-in site this Indexer uses; null for every other kind */
  port: IndexersPortPin | null;
  /**
   * Lower wins a tie
   * @minimum 1
   * @maximum 50
   */
  priority: number;
  /**
   * Seconds each request to it may take; null for the Node's default, 30
   * @minimum 10
   * @maximum 300
   * @nullable
   */
  queryTimeout: number | null;
  /** Its feed lists older uploads by offset, so RSS sync can page back to the last one it read. Read by its last Test */
  recentPaging: boolean;
  /** @minimum 1 */
  revision: number;
  rss: boolean;
  /** Its last RSS fetch; null before its first */
  rssFetch: IndexersRssFeed | null;
  /** By mode */
  searchModes: IndexersSearchMode[];
  searches: IndexersLimitUse;
  /**
   * How many background searches go to it at a time. A press of Search takes one more
   * @minimum 1
   * @maximum 10
   */
  searchesAtOnce: number;
  /** Its status in words */
  sentence: string;
  /** Its newest session on any Node, null for none */
  session: IndexersSession | null;
  /** It holds a login session, so it waits for its home rather than failing over unless allowFailover */
  signsIn: boolean;
  status: IndexersStatus;
  /** The Trackers it reaches */
  trackerIds: string[];
  /** Why its searches wait now, null when they don't */
  waits: IndexersOutcome | null;
}

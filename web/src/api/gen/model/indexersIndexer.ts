import type { IndexersIndexerKind } from "./indexersIndexerKind.ts";
import type { IndexersLimitUse } from "./indexersLimitUse.ts";
import type { IndexersOutcome } from "./indexersOutcome.ts";
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
  on: boolean;
  /**
   * Lower wins a tie
   * @minimum 1
   * @maximum 50
   */
  priority: number;
  /** @minimum 1 */
  revision: number;
  rss: boolean;
  /** By mode */
  searchModes: IndexersSearchMode[];
  searches: IndexersLimitUse;
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

import type { IndexersLimit } from "./indexersLimit.ts";
import type { IndexersNewIndexerKind } from "./indexersNewIndexerKind.ts";
import type { IndexersPastedCookie } from "./indexersPastedCookie.ts";

export interface IndexersNewIndexer {
  /**
   * A Torznab feed's base, to which SlipStream adds /api, or an RSS feed's whole address. Write-only
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?(\?[^#]*)?$
   */
  address: string;
  allowFailover?: boolean;
  /** Use an RSS feed whose items have no size */
  allowNoSize?: boolean;
  /**
   * A Torznab feed's. Write-only
   * @maxLength 256
   */
  apiKey?: string;
  automaticSearch?: boolean;
  /**
   * Newznab category ids; absent for every category it advertises
   * @nullable
   */
  categories?: number[] | null;
  /** A browser cookie for an RSS feed that needs one. Write-only */
  cookie?: IndexersPastedCookie;
  grabs?: IndexersLimit;
  /** A Node that holds Searching; absent for the default home */
  homeNodeId?: string;
  /** The kinds SlipStream can add so far */
  kind: IndexersNewIndexerKind;
  /**
   * The Content Types an RSS feed carries, in place of categories; required for an RSS feed
   * @nullable
   */
  media?: string[] | null;
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  on?: boolean;
  /**
   * @minimum 1
   * @maximum 50
   */
  priority?: number;
  rss?: boolean;
  searches?: IndexersLimit;
  /**
   * The Tracker an RSS feed reaches; required for an RSS feed
   * @maxLength 100
   */
  trackerName?: string;
}

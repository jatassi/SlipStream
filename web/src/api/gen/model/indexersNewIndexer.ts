import type { IndexersCaptchaAnswer } from "./indexersCaptchaAnswer.ts";
import type { IndexersLimit } from "./indexersLimit.ts";
import type { IndexersNewIndexerKind } from "./indexersNewIndexerKind.ts";
import type { IndexersNewIndexerValues } from "./indexersNewIndexerValues.ts";
import type { IndexersPastedCookie } from "./indexersPastedCookie.ts";

export interface IndexersNewIndexer {
  /**
   * A Torznab feed's base, to which SlipStream adds /api, or an RSS feed's whole address; required for both. Write-only
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?(\?[^#]*)?$
   */
  address?: string;
  allowFailover?: boolean;
  /** Use an RSS feed whose items have no size */
  allowNoSize?: boolean;
  /**
   * A Torznab feed's. Write-only
   * @maxLength 256
   */
  apiKey?: string;
  /** Absent for on */
  automaticSearch?: boolean;
  /**
   * One of the definition's addresses, absent for its first; or a native Indexer's address, one the port knows or a typed http or https one
   * @maxLength 2048
   */
  baseAddress?: string;
  /** The answer to a captcha loaded for this draft */
  captcha?: IndexersCaptchaAnswer;
  /**
   * Newznab category ids; absent for every category it advertises
   * @nullable
   */
  categories?: number[] | null;
  /**
   * The current two-factor code, for a site whose sign-in asks for one
   * @maxLength 16
   */
  code?: string;
  /** A browser cookie for an RSS feed that needs one, or for a catalog or native Indexer whose site signs in with one. Write-only */
  cookie?: IndexersPastedCookie;
  /**
   * A catalog Indexer's definition id; required for one
   * @maxLength 200
   */
  definition?: string;
  grabs?: IndexersLimit;
  /** A Node that holds Searching; absent for the default home */
  homeNodeId?: string;
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
  /** Absent for on */
  on?: boolean;
  /**
   * A native Indexer's port id; required for one
   * @maxLength 200
   */
  port?: string;
  /**
   * @minimum 1
   * @maximum 50
   */
  priority?: number;
  /** Absent for on */
  rss?: boolean;
  searches?: IndexersLimit;
  /**
   * The name of the Tracker the feed reaches when its items name no source; required for an RSS feed, absent for a Torznab feed's Indexer name
   * @maxLength 100
   */
  trackerName?: string;
  /** A catalog or native Indexer's settings by name, a checkbox's as true or false and a multiselect's as comma-separated option values; a setting left out takes its default. Sealed ones are write-only */
  values?: IndexersNewIndexerValues;
}

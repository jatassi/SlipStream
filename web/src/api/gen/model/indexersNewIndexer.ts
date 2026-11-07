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
  automaticSearch?: boolean;
  /**
   * One of the definition's addresses; absent for its first
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
  /** A browser cookie for an RSS feed that needs one, or for a catalog Indexer whose definition signs in with one. Write-only */
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
  on?: boolean;
  /**
   * @minimum 1
   * @maximum 50
   */
  priority?: number;
  rss?: boolean;
  searches?: IndexersLimit;
  /**
   * The name of the Tracker the feed reaches when its items name no source; required for an RSS feed, absent for a Torznab feed's Indexer name
   * @maxLength 100
   */
  trackerName?: string;
  /** A catalog Indexer's settings by name, a checkbox's as true or false; a setting left out takes its default. Sealed ones are write-only */
  values?: IndexersNewIndexerValues;
}

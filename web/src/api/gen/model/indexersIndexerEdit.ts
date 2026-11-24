import type { IndexersCaptchaAnswer } from "./indexersCaptchaAnswer.ts";
import type { IndexersIndexerEditValues } from "./indexersIndexerEditValues.ts";
import type { IndexersLimit } from "./indexersLimit.ts";
import type { IndexersPastedCookie } from "./indexersPastedCookie.ts";

export interface IndexersIndexerEdit {
  /**
   * Write-only
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?(\?[^#]*)?$
   */
  address?: string;
  /**
   * Null resets it to off
   * @nullable
   */
  allowFailover?: boolean | null;
  /** An RSS feed's */
  allowNoSize?: boolean;
  /**
   * Write-only
   * @maxLength 256
   */
  apiKey?: string;
  /**
   * Null resets it to on
   * @nullable
   */
  automaticSearch?: boolean | null;
  /**
   * A catalog Indexer's: one of its definition's addresses. A built-in site's: one the site lists or a typed http or https one
   * @maxLength 2048
   */
  baseAddress?: string;
  /** The answer to a captcha loaded for this Indexer, for the Test the change runs */
  captcha?: IndexersCaptchaAnswer;
  /**
   * Null for every category it advertises
   * @nullable
   */
  categories?: number[] | null;
  /**
   * The current two-factor code, for the Test the change runs
   * @maxLength 16
   */
  code?: string;
  /** A new browser cookie. Write-only */
  cookie?: IndexersPastedCookie;
  /** Null turns the limit off */
  grabs?: IndexersLimit | null;
  /** A Node with the Searching Role */
  homeNodeId?: string;
  /** An RSS feed's Content Types */
  media?: string[];
  /**
   * @minLength 1
   * @maxLength 100
   */
  name?: string;
  /**
   * Null resets it to on
   * @nullable
   */
  on?: boolean | null;
  /**
   * Null resets it to 25
   * @minimum 1
   * @maximum 50
   * @nullable
   */
  priority?: number | null;
  /**
   * Seconds; null resets it to the Node's default, 30
   * @minimum 10
   * @maximum 300
   * @nullable
   */
  queryTimeout?: number | null;
  /**
   * Null resets it to on
   * @nullable
   */
  rss?: boolean | null;
  /** Null turns the limit off */
  searches?: IndexersLimit | null;
  /**
   * Null resets it to 2
   * @minimum 1
   * @maximum 10
   * @nullable
   */
  searchesAtOnce?: number | null;
  /** A catalog or built-in site Indexer's settings to change, by name; a secret one sent empty keeps its saved value */
  values?: IndexersIndexerEditValues;
}

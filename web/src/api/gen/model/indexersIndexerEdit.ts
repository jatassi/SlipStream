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
   * Null for every category it advertises
   * @nullable
   */
  categories?: number[] | null;
  /** A new browser cookie for an RSS feed. Write-only */
  cookie?: IndexersPastedCookie;
  /** Null turns the limit off */
  grabs?: IndexersLimit | null;
  /** A Node that holds Searching */
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
   * Null resets it to on
   * @nullable
   */
  rss?: boolean | null;
  /** Null turns the limit off */
  searches?: IndexersLimit | null;
}

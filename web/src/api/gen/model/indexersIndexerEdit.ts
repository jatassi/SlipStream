import type { IndexersLimit } from "./indexersLimit.ts";

export interface IndexersIndexerEdit {
  /**
   * Write-only
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?$
   */
  address?: string;
  /**
   * Null resets it to off
   * @nullable
   */
  allowFailover?: boolean | null;
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
  /** Null turns the limit off */
  grabs?: IndexersLimit | null;
  /** A Node that holds Searching */
  homeNodeId?: string;
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

import type { IndexersLimit } from "./indexersLimit.ts";
import type { IndexersNewIndexerKind } from "./indexersNewIndexerKind.ts";

export interface IndexersNewIndexer {
  /**
   * The Torznab feed's base; SlipStream adds /api. Write-only
   * @maxLength 2048
   * @pattern ^https?://[^/?#]+(/[^?#]*)?$
   */
  address: string;
  allowFailover?: boolean;
  /**
   * Write-only
   * @maxLength 256
   */
  apiKey: string;
  automaticSearch?: boolean;
  /**
   * Newznab category ids; absent for every category it advertises
   * @nullable
   */
  categories?: number[] | null;
  grabs?: IndexersLimit;
  /** A Node that holds Searching; absent for the default home */
  homeNodeId?: string;
  /** The kinds SlipStream can add so far */
  kind: IndexersNewIndexerKind;
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
}

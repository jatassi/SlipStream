import type { IndexersLimit } from "./indexersLimit.ts";

export interface IndexersPortFacts {
  /** A new Indexer's grab limit unless the form sets one */
  grabs: IndexersLimit;
  /** The site lists its new uploads, so RSS sync can use it */
  recent: boolean;
  /** A new Indexer's search limit unless the form sets one */
  searches: IndexersLimit;
  /** The least time between two requests to the site */
  spacingSeconds: number;
  /**
   * When the Prowlarr commit SlipStream's support for the site was synced to was made
   * @nullable
   */
  synced: string | null;
  /** The pasted cookie works only with the User-Agent of the browser it came from */
  userAgent: boolean;
}

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
   * The date of the Prowlarr commit that SlipStream's support for this site is based on
   * @nullable
   */
  synced: string | null;
  /** The pasted cookie works only with the User-Agent of the browser it came from */
  userAgent: boolean;
}

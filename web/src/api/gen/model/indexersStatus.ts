import type { IndexersFailure } from "./indexersFailure.ts";
import type { IndexersStatusCause } from "./indexersStatusCause.ts";
import type { IndexersStatusState } from "./indexersStatusState.ts";

export interface IndexersStatus {
  /**
   * Null unless stopped or home_only
   * @nullable
   */
  cause: IndexersStatusCause;
  lastFailure: IndexersFailure | null;
  /**
   * The Node the stop was met on
   * @nullable
   */
  nodeId: string | null;
  /**
   * The backoff ladder's position, stepping down one on each success
   * @minimum 0
   */
  rung: number;
  /**
   * When it began failing, stopped or began waiting for home
   * @nullable
   */
  since: string | null;
  /** A challenge_unsolved that the Node's Challenge solver couldn't clear */
  solverTried: boolean;
  /** home_only: the Indexer stopped while searching from a failover Node, and searches only from its home Node until that Node is back */
  state: IndexersStatusState;
  /**
   * When the site last responded successfully
   * @nullable
   */
  successAt: string | null;
  /**
   * The site's or feed's own words
   * @nullable
   */
  text: string | null;
  /**
   * When SlipStream sends a limited or failing Indexer its next request
   * @nullable
   */
  until: string | null;
}

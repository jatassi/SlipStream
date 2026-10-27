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
  /** home_only: a stop met on a failover Node, which waits for home */
  state: IndexersStatusState;
  /**
   * The last answer the site responded to
   * @nullable
   */
  successAt: string | null;
  /**
   * The site's or feed's own words
   * @nullable
   */
  text: string | null;
  /**
   * When a limited or failing Indexer is asked again
   * @nullable
   */
  until: string | null;
}

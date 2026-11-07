import type { IndexersOutcomeKind } from "./indexersOutcomeKind.ts";

export interface IndexersOutcome {
  /**
   * Set for stopped
   * @nullable
   */
  cause: string | null;
  kind: IndexersOutcomeKind;
  /** Waiting past the offline grace */
  lapsed: boolean;
  /**
   * Set for waiting: the Node waited for; for stopped: the Node the stop was met on
   * @nullable
   */
  nodeId: string | null;
  /** @minimum 0 */
  releases: number;
  /**
   * Set for waiting: since when the home Node is away; null when it holds no Role to search with
   * @nullable
   */
  since: string | null;
  /**
   * Set for stopped: the site's own words
   * @nullable
   */
  text: string | null;
  /**
   * Set for limited, failing and limit_reached
   * @nullable
   */
  until: string | null;
}

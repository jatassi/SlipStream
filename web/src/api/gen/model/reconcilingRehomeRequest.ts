import type { ReconcilingSaveRoot } from "./reconcilingSaveRoot.ts";

export interface ReconcilingRehomeRequest {
  /** Whether this ends the step, so SlipStream stops waiting for the old Controller's folders */
  done: boolean;
  /**
   * Each library root to save. A root left out stays for later: no scan reads it and Reconciling waits on it
   * @maxItems 1000
   */
  roots: ReconcilingSaveRoot[];
}

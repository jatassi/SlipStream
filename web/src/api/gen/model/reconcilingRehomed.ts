import type { ReconcilingSavedRoot } from "./reconcilingSavedRoot.ts";

export interface ReconcilingRehomed {
  /** How each root's save went, in the order asked */
  roots: ReconcilingSavedRoot[];
}

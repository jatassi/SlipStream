import type { RootsRefusalReason } from "./rootsRefusalReason.ts";

export interface RootsRefusal {
  /** Why the folder can't serve its purpose */
  reason: RootsRefusalReason;
  /** The root the folder is inside, contains or carries the marker of */
  rootId?: string;
}

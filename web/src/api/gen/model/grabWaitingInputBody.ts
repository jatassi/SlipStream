import type { GrabsCellKey } from "./grabsCellKey.ts";

export interface GrabWaitingInputBody {
  /**
   * The waiting Items and Targets to grab now; left out, every waiting one of the Work
   * @nullable
   */
  cells?: GrabsCellKey[] | null;
}

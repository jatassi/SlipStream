import type { GrabsCellKey } from "./grabsCellKey.ts";

export interface GrabWaitingInputBody {
  /**
   * The waiting cells to grab now; left out, every waiting cell of the Work
   * @nullable
   */
  cells?: GrabsCellKey[] | null;
}

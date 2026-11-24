import type { MovesDropped } from "./movesDropped.ts";

export interface MovesStartedMoves {
  batchId: string;
  /** The preview's units that can't move now; empty for a Start sent again */
  dropped: MovesDropped[];
  /**
   * The units the batch moves
   * @minimum 0
   */
  started: number;
}

import type { MovesMoveAskOrigin } from "./movesMoveAskOrigin.ts";
import type { MovesWorkTo } from "./movesWorkTo.ts";

export interface MovesMoveAsk {
  /** Where the Admin starts it: a Work's page, or the Library's bulk bar */
  origin: MovesMoveAskOrigin;
  /**
   * @minItems 1
   * @maxItems 10000
   */
  works: MovesWorkTo[];
}

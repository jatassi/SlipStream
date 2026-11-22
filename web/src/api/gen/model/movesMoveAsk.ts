import type { MovesMoveAskOrigin } from "./movesMoveAskOrigin.ts";
import type { MovesTorrentTo } from "./movesTorrentTo.ts";
import type { MovesWorkTo } from "./movesWorkTo.ts";

export interface MovesMoveAsk {
  /** Where the Admin starts it: a Work's page, the Library's bulk bar, or the Torrent table */
  origin: MovesMoveAskOrigin;
  /**
   * @maxItems 10000
   * @nullable
   */
  torrents?: MovesTorrentTo[] | null;
  /**
   * At least one Work or torrent
   * @maxItems 10000
   */
  works: MovesWorkTo[];
}

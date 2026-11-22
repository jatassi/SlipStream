import type { MovesClientTo } from "./movesClientTo.ts";
import type { MovesMoveAskOrigin } from "./movesMoveAskOrigin.ts";
import type { MovesMoveOptions } from "./movesMoveOptions.ts";
import type { MovesRootTo } from "./movesRootTo.ts";
import type { MovesTorrentTo } from "./movesTorrentTo.ts";
import type { MovesWorkTo } from "./movesWorkTo.ts";

export interface MovesMoveAsk {
  /**
   * @maxItems 100
   * @nullable
   */
  clients?: MovesClientTo[] | null;
  options?: MovesMoveOptions;
  /** Where the Admin starts it: a Work's page, the Library's bulk bar, a root's row, a Node's drain, a Download Client, or the Torrent table */
  origin: MovesMoveAskOrigin;
  /**
   * @maxItems 100
   * @nullable
   */
  roots?: MovesRootTo[] | null;
  /**
   * @maxItems 10000
   * @nullable
   */
  torrents?: MovesTorrentTo[] | null;
  /**
   * At least one Work, root, torrent or client
   * @maxItems 10000
   */
  works: MovesWorkTo[];
}

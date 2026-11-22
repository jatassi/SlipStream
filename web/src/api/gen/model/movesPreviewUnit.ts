import type { MovesPlace } from "./movesPlace.ts";
import type { MovesPreviewUnitKind } from "./movesPreviewUnitKind.ts";
import type { MovesPreviewUnitVerdict } from "./movesPreviewUnitVerdict.ts";
import type { MovesReason } from "./movesReason.ts";

export interface MovesPreviewUnit {
  /**
   * Bytes that copy; a hard link or a clone copies none
   * @minimum 0
   */
  bytes: number;
  from: MovesPlace;
  /** Stable per unit inside the preview, such as work:<id> or torrent:<id> */
  key: string;
  kind: MovesPreviewUnitKind;
  name: string;
  /** Each check worse than green */
  reasons: MovesReason[];
  to: MovesPlace;
  /** A torrent unit's */
  torrentId?: string;
  /** green moves, amber moves with a note, red stays */
  verdict: MovesPreviewUnitVerdict;
  /** A Work unit's; the zero id for a torrent */
  workId: string;
}

import type { MovesFailure } from "./movesFailure.ts";
import type { MovesMoveUnitKind } from "./movesMoveUnitKind.ts";
import type { MovesMoveUnitPhase } from "./movesMoveUnitPhase.ts";
import type { MovesMoveUnitWaiting } from "./movesMoveUnitWaiting.ts";
import type { MovesPlace } from "./movesPlace.ts";

export interface MovesMoveUnit {
  batchId: string;
  /** @minimum 0 */
  bytes: number;
  contentType: string;
  /** Why it failed; null unless it failed */
  failure: MovesFailure | null;
  from: MovesPlace;
  id: string;
  kind: MovesMoveUnitKind;
  name: string;
  /** What it recorded for the Admin, as sentences */
  notes: string[];
  phase: MovesMoveUnitPhase;
  /** Whether Switch now can end its wait */
  switchNow: boolean;
  to: MovesPlace;
  /** A torrent unit's */
  torrentId?: string;
  /** A failed or canceled Move removed what it placed */
  undone: boolean;
  /** What it waits for; absent while it waits for nothing */
  waiting?: MovesMoveUnitWaiting;
  waitingSince?: string;
  /** What it does now, such as Copying to den */
  words: string;
  /** A Work unit's; the zero id for a torrent */
  workId: string;
}

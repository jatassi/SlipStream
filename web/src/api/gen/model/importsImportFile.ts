import type { ImportsAction } from "./importsAction.ts";
import type { ImportsCellRef } from "./importsCellRef.ts";
import type { ImportsFileHold } from "./importsFileHold.ts";
import type { ImportsForceOption } from "./importsForceOption.ts";
import type { ImportsImportFileOutcome } from "./importsImportFileOutcome.ts";
import type { ImportsImportFilePlacedBy } from "./importsImportFilePlacedBy.ts";
import type { ImportsImportFileQuality } from "./importsImportFileQuality.ts";
import type { ImportsReason } from "./importsReason.ts";
import type { ImportsTargetVerdict } from "./importsTargetVerdict.ts";

export interface ImportsImportFile {
  /** The other actions its row offers */
  actions: ImportsAction[];
  /** The key of the cut its name carries */
  cut: string;
  /**
   * The library file it became; null until it is placed
   * @nullable
   */
  fileId: string | null;
  /** The Items and Targets it is the file for, or will be once confirmed */
  fills: ImportsCellRef[];
  /** Each Target it can be forced into, with the files that deletes */
  forces: ImportsForceOption[];
  /** What it waits on that SlipStream retries automatically; null while nothing holds it */
  hold: ImportsFileHold | null;
  /** The Items of the Work it holds, in tree order; empty until a Work places it */
  itemIds: string[];
  /** Move files can move it, because no torrent SlipStream tracks is seeding it */
  movable: boolean;
  outcome: ImportsImportFileOutcome;
  /** For a root Import relative to the root, / separated */
  path: string;
  /**
   * How it went into the library; null until it is placing
   * @nullable
   */
  placedBy: ImportsImportFilePlacedBy;
  /**
   * The file's place in its Import, which names it in actions
   * @minimum 0
   */
  position: number;
  /**
   * The quality its name and media info read as; null when ffprobe can't read it
   * @nullable
   */
  quality: ImportsImportFileQuality;
  /** Why it is left out, waits, isn't better or failed; null otherwise */
  reason: ImportsReason | null;
  /** The action its row offers first; null when it offers none */
  recommended: ImportsAction | null;
  /** @minimum 0 */
  size: number;
  /**
   * Bytes its holding or staged copy uses while it waits
   * @minimum 0
   */
  staged: number;
  /** Each Target's verdict on it, for its Details */
  verdicts: ImportsTargetVerdict[];
}

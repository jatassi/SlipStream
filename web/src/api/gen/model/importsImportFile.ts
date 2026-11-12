import type { ImportsImportFileOutcome } from "./importsImportFileOutcome.ts";
import type { ImportsImportFileQuality } from "./importsImportFileQuality.ts";
import type { ImportsReason } from "./importsReason.ts";

export interface ImportsImportFile {
  /** The key of the cut its name carries */
  cut: string;
  /** The Items of the Work it holds, in tree order; empty until a Work places it */
  itemIds: string[];
  outcome: ImportsImportFileOutcome;
  /** For a root Import relative to the root, / separated */
  path: string;
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
  /** Why it waits; null once it's placed */
  reason: ImportsReason | null;
  /** @minimum 0 */
  size: number;
}

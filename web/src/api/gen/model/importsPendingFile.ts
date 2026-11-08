import type { ImportsPendingFileQuality } from "./importsPendingFileQuality.ts";
import type { ImportsReason } from "./importsReason.ts";

export interface ImportsPendingFile {
  /** The key of the cut its name carries */
  cut: string;
  /** The Items of the suggested Work its path names; empty until a Work places it */
  itemIds: string[];
  /** Relative to the root, / separated */
  path: string;
  /**
   * The quality its name and media info read as; null when ffprobe can't read it
   * @nullable
   */
  quality: ImportsPendingFileQuality;
  reason: ImportsReason;
  /** @minimum 0 */
  size: number;
}

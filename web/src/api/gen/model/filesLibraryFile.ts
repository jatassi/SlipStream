import type { FilesLibraryFileQuality } from "./filesLibraryFileQuality.ts";
import type { FilesProbe } from "./filesProbe.ts";

export interface FilesLibraryFile {
  id: string;
  /** Relative to its root, / separated */
  path: string;
  probe: FilesProbe;
  quality: FilesLibraryFileQuality;
  /** @minimum 0 */
  size: number;
}

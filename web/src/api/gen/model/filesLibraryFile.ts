import type { FilesProbe } from "./filesProbe.ts";

export interface FilesLibraryFile {
  id: string;
  /** Relative to its root, / separated */
  path: string;
  probe: FilesProbe;
  /** @minimum 0 */
  size: number;
}

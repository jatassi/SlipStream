import type { FilesFill } from "./filesFill.ts";
import type { FilesListedFileQuality } from "./filesListedFileQuality.ts";
import type { FilesProbe } from "./filesProbe.ts";

export interface FilesListedFile {
  /** The cells it fills; empty for a Kept file */
  fills: FilesFill[];
  /** The folder it sits in, relative to its root, / separated */
  folder: string;
  id: string;
  /** A Kept file, which fills no cell */
  kept: boolean;
  /** Relative to its root, / separated */
  path: string;
  probe: FilesProbe;
  quality: FilesListedFileQuality;
  /** @minimum 0 */
  size: number;
}

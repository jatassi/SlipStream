import type { FilesFill } from "./filesFill.ts";
import type { FilesListedFileQuality } from "./filesListedFileQuality.ts";
import type { FilesMediaInfo } from "./filesMediaInfo.ts";
import type { FilesVideoCodec } from "./filesVideoCodec.ts";

export interface FilesListedFile {
  /** Its video codec in the words a Target filters on; null for a codec they have no word for */
  codec: FilesVideoCodec | null;
  /** The key of its own cut */
  cut: string;
  /** The cells it fills; empty for a Kept or gone file */
  fills: FilesFill[];
  /** The folder it sits in, relative to its root, / separated */
  folder: string;
  /**
   * When a library scan found it missing from its root; null while it is there
   * @nullable
   */
  goneAt: string | null;
  id: string;
  /** A Kept file, which fills no cell */
  kept: boolean;
  /** The Target name its file name carries, before any suffix */
  label: string;
  /** What ffprobe read of it; null for a file read before SlipStream kept its tracks */
  media: FilesMediaInfo | null;
  /**
   * Its modification time as a library scan or its import last read it; null until one did
   * @nullable
   */
  modifiedAt: string | null;
  /** Relative to its root, / separated */
  path: string;
  /**
   * The first line ffprobe printed when the last library scan couldn't read it, and media is then the read before; null when that read worked
   * @nullable
   */
  probeError: string | null;
  quality: FilesListedFileQuality;
  /** @minimum 0 */
  size: number;
}

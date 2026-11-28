import type { FilesFileGrab } from "./filesFileGrab.ts";
import type { FilesFill } from "./filesFill.ts";
import type { FilesListedFilePlacedBy } from "./filesListedFilePlacedBy.ts";
import type { FilesListedFileQuality } from "./filesListedFileQuality.ts";
import type { FilesMadeFrom } from "./filesMadeFrom.ts";
import type { FilesMediaInfo } from "./filesMediaInfo.ts";
import type { FilesVideoCodec } from "./filesVideoCodec.ts";

export interface FilesListedFile {
  /** Its video codec in the words a Target filters on; null for a codec they have no word for */
  codec: FilesVideoCodec | null;
  /** The key of its own cut */
  cut: string;
  /** The Items and Targets it is the file for; empty for a Kept or gone file */
  fills: FilesFill[];
  /** The folder it sits in, relative to its root, / separated */
  folder: string;
  /**
   * When a library scan found it missing from its root; null while it is there
   * @nullable
   */
  goneAt: string | null;
  /** The Grab whose download it came from; null for a file no Grab brought */
  grab: FilesFileGrab | null;
  id: string;
  /** A Kept file, which no Target uses */
  kept: boolean;
  /** The Target name its file name carries, before any suffix */
  label: string;
  /** The file a Conversion made it from; null for a file no Conversion made */
  madeFrom: FilesMadeFrom | null;
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
   * How its import put it into the library; null for a file no import recorded
   * @nullable
   */
  placedBy: FilesListedFilePlacedBy;
  /**
   * The first line ffprobe printed when the last library scan couldn't read the file, in which case media is from the read before; null when the last read worked
   * @nullable
   */
  probeError: string | null;
  quality: FilesListedFileQuality;
  /** @minimum 0 */
  size: number;
}

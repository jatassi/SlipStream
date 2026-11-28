import type { FilesConversionFailedNeedClass } from "./filesConversionFailedNeedClass.ts";
import type { FilesConversionFailedNeedKind } from "./filesConversionFailedNeedKind.ts";

export interface FilesConversionFailedNeed {
  /** run: it failed on every Node it may try; source: its source file can't be converted */
  class: FilesConversionFailedNeedClass;
  /** The fault, such as encoder_failed */
  code: string;
  conversionId: string;
  /** any under any cut, else the key of the cut these Items need at this Target */
  edition: string;
  /** A run of Items in tree order whose entries match */
  itemIds: string[];
  kind: FilesConversionFailedNeedKind;
  /** The Target whose made file failed */
  targetId: string;
}

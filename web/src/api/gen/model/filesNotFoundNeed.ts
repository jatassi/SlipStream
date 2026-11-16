import type { FilesNotFoundNeedKind } from "./filesNotFoundNeedKind.ts";

export interface FilesNotFoundNeed {
  /** any under any cut, else the key of the cut these Items need at this Target */
  edition: string;
  /** A run of Items in tree order whose entries match */
  itemIds: string[];
  /** Missing Items whose latest search found no Release, or none acceptable. It shows on the Work page only, and the Library's Needs you count leaves it out */
  kind: FilesNotFoundNeedKind;
  /** When the latest search finished */
  lastSearchedAt: string;
  /**
   * When SlipStream searches them next; null while nothing will
   * @nullable
   */
  nextSearchAt: string | null;
}

import type { FilesFailedNeedKind } from "./filesFailedNeedKind.ts";

export interface FilesFailedNeed {
  /** any under any cut, else the key of the cut the cells want */
  edition: string;
  /** The Grab that failed */
  grabId: string;
  /** A run of Items in tree order whose entries match */
  itemIds: string[];
  kind: FilesFailedNeedKind;
  /** The Target whose download failed */
  targetId: string;
  /** The Targets of the same Items and Edition, still missing, that wait on the failed one */
  waiting: string[];
}

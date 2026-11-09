import type { FilesRowBlockReason } from "./filesRowBlockReason.ts";

export interface FilesRowBlock {
  /** Why the file can't be renamed now, then what to do */
  detail: string;
  /** For clash, the Work whose file ends at the same path */
  otherWorkId?: string;
  reason: FilesRowBlockReason;
}

import type { FilesUnmatchedType } from "./filesUnmatchedType.ts";

export interface UnmatchedOutputBody {
  /** Each Content Type with such a file */
  contentTypes: FilesUnmatchedType[];
}

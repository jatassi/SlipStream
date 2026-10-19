import type { RootsMissingFileReason } from "./rootsMissingFileReason.ts";

export interface RootsMissingFile {
  /** Where the file records its bytes, relative to the root and / separated */
  path: string;
  /** missing: nothing is at that path; size_differs: a file is there at another size */
  reason: RootsMissingFileReason;
}

import type { WorksLibraryRow } from "./worksLibraryRow.ts";

export interface WorksSearchResult {
  /** The rows of the first limit of them, in the same order */
  rows: WorksLibraryRow[];
  /** Every matching Work, best first */
  workIds: string[];
}

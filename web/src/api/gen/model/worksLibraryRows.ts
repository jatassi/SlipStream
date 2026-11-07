import type { WorksLibraryRow } from "./worksLibraryRow.ts";

export interface WorksLibraryRows {
  /** By sort label, then id */
  rows: WorksLibraryRow[];
}

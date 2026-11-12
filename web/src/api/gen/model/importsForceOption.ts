import type { ImportsCellRef } from "./importsCellRef.ts";

export interface ImportsForceOption {
  cell: ImportsCellRef;
  /** The paths of the library files the force deletes, relative to their root */
  deletes: string[];
}

import type { ImportsCellRef } from "./importsCellRef.ts";
import type { ImportsForceFileAction } from "./importsForceFileAction.ts";

export interface ImportsForceFile {
  /** Put the file into a Target even though it isn't an upgrade there */
  action: ImportsForceFileAction;
  cell: ImportsCellRef;
}

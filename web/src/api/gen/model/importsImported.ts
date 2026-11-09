import type { ImportsEntry } from "./importsEntry.ts";

export interface ImportsImported {
  /** The files left waiting; null once none is */
  entry: ImportsEntry | null;
}

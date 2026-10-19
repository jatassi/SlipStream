import type { RootsMissingFile } from "./rootsMissingFile.ts";

export interface RootsMissingFiles {
  /** The recorded files not at the new place */
  count: number;
  /** Up to the first 20 of them, in file order */
  first: RootsMissingFile[];
}

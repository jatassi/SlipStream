import type { FilesState } from "./filesState.ts";

export interface FilesItemStatus {
  itemId: string;
  /** The first state its cells hold, in the glossary's order; null for an Item with no cell */
  status: FilesState | null;
}

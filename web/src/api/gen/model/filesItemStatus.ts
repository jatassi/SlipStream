import type { FilesItemStatusStatus } from "./filesItemStatusStatus.ts";

export interface FilesItemStatus {
  itemId: string;
  /**
   * The first state its cells hold, in the glossary's order; null for an Item with no cell
   * @nullable
   */
  status: FilesItemStatusStatus;
}

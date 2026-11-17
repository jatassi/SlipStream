import type { HistoryListEntry } from "./historyListEntry.ts";

export interface HistoryListed {
  /** Newest first */
  entries: HistoryListEntry[];
  /**
   * The before of the next page; null on the last page
   * @nullable
   */
  next: number | null;
}

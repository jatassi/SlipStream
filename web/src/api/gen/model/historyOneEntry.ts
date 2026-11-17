import type { HistoryFile } from "./historyFile.ts";
import type { HistoryHealthItem } from "./historyHealthItem.ts";
import type { HistoryListEntry } from "./historyListEntry.ts";
import type { HistoryStageAt } from "./historyStageAt.ts";

export interface HistoryOneEntry {
  entry: HistoryListEntry;
  /** Each file an Import placed */
  files: HistoryFile[];
  /** A Health entry's items */
  health: HistoryHealthItem[];
  /** Whole sentences, with clock times in zone */
  lines: string[];
  /** Each stage a failed Grab reached, in order */
  stages: HistoryStageAt[];
}

import type { HistoryLinks } from "./historyLinks.ts";
import type { HistoryListEntryGroup } from "./historyListEntryGroup.ts";
import type { HistoryListEntryTone } from "./historyListEntryTone.ts";
import type { HistoryWorkSummary } from "./historyWorkSummary.ts";

export interface HistoryListEntry {
  /** When it was captured */
  at: string;
  group: HistoryListEntryGroup;
  id: number;
  /** A kind GET /history/kinds lists */
  kind: string;
  links: HistoryLinks;
  tone: HistoryListEntryTone;
  /** Such as Imported 10 episodes of Severance */
  words: string;
  /** The one Work it is about; null for none or many, and once that Work is deleted */
  work: HistoryWorkSummary | null;
}

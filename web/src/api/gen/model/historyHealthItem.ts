import type { HistoryHealthItemLevel } from "./historyHealthItemLevel.ts";

export interface HistoryHealthItem {
  level: HistoryHealthItemLevel;
  subject: string;
  text: string;
}

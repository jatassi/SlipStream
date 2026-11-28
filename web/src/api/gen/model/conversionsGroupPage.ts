import type { ConversionsConversionRow } from "./conversionsConversionRow.ts";

export interface ConversionsGroupPage {
  /**
   * null on the last page
   * @nullable
   */
  cursor: string | null;
  /** In run order */
  rows: ConversionsConversionRow[];
}

import type { ConversionsConversionRow } from "./conversionsConversionRow.ts";
import type { ConversionsCounts } from "./conversionsCounts.ts";
import type { ConversionsLaneRows } from "./conversionsLaneRows.ts";

export interface ConversionsConversionList {
  /** Failed Conversions, newest first */
  attention: ConversionsConversionRow[];
  /** Running and paused Conversions */
  converting: ConversionsConversionRow[];
  counts: ConversionsCounts;
  /**
   * Pages Up next; null on its last page
   * @nullable
   */
  cursor: string | null;
  /** Up next, one entry per lane in run order */
  next: ConversionsLaneRows[];
}

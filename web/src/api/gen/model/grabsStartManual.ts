import type { GrabsManualCell } from "./grabsManualCell.ts";

export interface GrabsStartManual {
  /** Start a new search even when the Work's Manual search has this scope and query */
  again?: boolean;
  /** One cell to plan for; send it without levelIds and itemIds */
  cell?: GrabsManualCell;
  /** @maxItems 20000 */
  itemIds?: string[];
  /**
   * Each level stands for every Item under it
   * @maxItems 20000
   */
  levelIds?: string[];
  /**
   * Text every Indexer searches for in place of the Item's terms; only for a scope of one Item
   * @maxLength 200
   */
  query?: string;
}

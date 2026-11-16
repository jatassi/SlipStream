import type { ListWantedReasonItem } from "./listWantedReasonItem.ts";
import type { ListWantedSort } from "./listWantedSort.ts";

export type ListWantedParams = {
  /**
   * Left out, every Content Type
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  /**
   * Left out, every reason but gone
   * @nullable
   */
  reason?: ListWantedReasonItem[] | null;
  /**
   * next is the next Backlog search first, last the longest since the last search first
   */
  sort?: ListWantedSort;
  /**
   * The next value of the page before; left out, the first page
   */
  cursor?: string;
  /**
   * @minimum 1
   * @maximum 100
   */
  limit?: number;
};

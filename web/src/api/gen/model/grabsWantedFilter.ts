import type { GrabsWantedFilterReasonsItem } from "./grabsWantedFilterReasonsItem.ts";

export interface GrabsWantedFilter {
  /**
   * Left out, every Content Type
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  /**
   * Left out, every reason but gone
   * @nullable
   */
  reasons?: GrabsWantedFilterReasonsItem[] | null;
}

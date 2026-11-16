import type { GrabsWantedFilter } from "./grabsWantedFilter.ts";

export interface GrabsSeveralBody {
  /** Every Work Wanted lists for this filter; send this or workIds */
  wanted?: GrabsWantedFilter;
  /**
   * The Works to search; send this or wanted
   * @nullable
   */
  workIds?: string[] | null;
}

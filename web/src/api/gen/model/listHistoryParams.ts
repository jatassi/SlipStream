import type { ListHistoryGroupsItem } from "./listHistoryGroupsItem.ts";

export type ListHistoryParams = {
  /**
   * Only entries with a smaller id: the next member of the previous page; left out, the newest
   * @minimum 1
   */
  before?: number;
  /**
   * @minimum 1
   * @maximum 100
   */
  limit?: number;
  /**
   * Only entries about this Content Type's Works; left out, every entry, Health included
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  /**
   * Only these groups; left out, all eight
   * @nullable
   */
  groups?: ListHistoryGroupsItem[] | null;
  /**
   * A comma list of the kinds GET /history/kinds lists. Each kind narrows its own group, and groups, when given, must include that group
   * @nullable
   */
  kinds?: string[] | null;
  /**
   * Only entries captured at or after this instant
   */
  since?: string;
  /**
   * Only entries about this Work
   */
  work?: string;
};

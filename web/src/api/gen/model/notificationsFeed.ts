import type { NotificationsFeedRow } from "./notificationsFeedRow.ts";

export interface NotificationsFeed {
  /**
   * The cursor for the next page; null at the end
   * @nullable
   */
  before: number | null;
  /** Newest first */
  rows: NotificationsFeedRow[];
  /**
   * How many rows are unread, at most 100; null for a request made with the API key
   * @nullable
   */
  unread: number | null;
}

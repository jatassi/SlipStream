import type { NotificationsStatusShown } from "./notificationsStatusShown.ts";

export interface NotificationsStatus {
  /** Messages given up since the connection was last saved or tested, because they waited too long or the provider refused them. */
  givenUp: number;
  /** @nullable */
  lastSent: string | null;
  /** Why it isn't sending, worded when read. Empty when it is. */
  reason: string;
  shown: NotificationsStatusShown;
  /**
   * When it started failing, waiting or paused; null otherwise
   * @nullable
   */
  since: string | null;
  /**
   * When a waiting connection sends again; null otherwise
   * @nullable
   */
  until: string | null;
  /** Messages waiting to be sent. */
  waiting: number;
}

import type { NotificationsEditBodySecrets } from "./notificationsEditBodySecrets.ts";
import type { NotificationsEditBodyValues } from "./notificationsEditBodyValues.ts";

export interface NotificationsEditBody {
  /** @nullable */
  events?: string[] | null;
  /** @nullable */
  name?: string | null;
  /**
   * A Custom Script's Node id
   * @nullable
   */
  node?: string | null;
  /** @nullable */
  on?: boolean | null;
  /** @nullable */
  posters?: boolean | null;
  /** A secret to replace. Left out or empty keeps it; null removes an optional one. Write-only */
  secrets?: NotificationsEditBodySecrets;
  /** Settings to change, by key; the others keep their values */
  values?: NotificationsEditBodyValues;
  /** @nullable */
  warnings?: boolean | null;
}

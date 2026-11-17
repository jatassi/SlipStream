import type { NotificationsConnectionShown } from "./notificationsConnectionShown.ts";
import type { NotificationsConnectionValues } from "./notificationsConnectionValues.ts";
import type { NotificationsStatus } from "./notificationsStatus.ts";

export interface NotificationsConnection {
  events: string[];
  id: string;
  name: string;
  /**
   * The Node a Custom Script runs on; null for every other provider
   * @nullable
   */
  node: string | null;
  on: boolean;
  posters: boolean;
  provider: string;
  /** The secret fields that have a value. Secrets are never returned */
  sealed: string[];
  /** A part of a secret value that is safe to show, such as a webhook URL's host */
  shown: NotificationsConnectionShown;
  status: NotificationsStatus;
  /** The provider's settings that aren't secret */
  values: NotificationsConnectionValues;
  /** Whether it gets Health warnings as well as errors */
  warnings: boolean;
}

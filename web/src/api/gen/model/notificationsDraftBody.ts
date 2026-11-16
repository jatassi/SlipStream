import type { NotificationsDraftBodySecrets } from "./notificationsDraftBodySecrets.ts";
import type { NotificationsDraftBodyValues } from "./notificationsDraftBodyValues.ts";

export interface NotificationsDraftBody {
  /**
   * The events it gets; left out, the recommended ones
   * @nullable
   */
  events?: string[] | null;
  /** @maxLength 100 */
  name: string;
  /** The Node a Custom Script runs on; left out for every other provider */
  node?: string;
  on: boolean;
  posters: boolean;
  /** One of the ids listNotificationProviders lists */
  provider: string;
  /** The secret settings by key. Write-only */
  secrets?: NotificationsDraftBodySecrets;
  /** The provider's settings that aren't secret, by key */
  values?: NotificationsDraftBodyValues;
  warnings: boolean;
}

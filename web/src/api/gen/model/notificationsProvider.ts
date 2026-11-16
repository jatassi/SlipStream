import type { NotificationsField } from "./notificationsField.ts";
import type { NotificationsProviderImage } from "./notificationsProviderImage.ts";

export interface NotificationsProvider {
  /** In the order the sheet shows them */
  fields: NotificationsField[];
  id: string;
  /** How the provider takes a poster */
  image: NotificationsProviderImage;
  label: string;
}

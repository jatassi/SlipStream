import type { NotificationsFieldKind } from "./notificationsFieldKind.ts";
import type { NotificationsOption } from "./notificationsOption.ts";

export interface NotificationsField {
  default?: unknown;
  /** Shown under Details in the sheet */
  details: boolean;
  help: string;
  key: string;
  kind: NotificationsFieldKind;
  label: string;
  /** A number's highest value, or a duration's, in seconds */
  max?: number;
  /** A number's lowest value, or a duration's, in seconds */
  min?: number;
  optional: boolean;
  /**
   * A select's choices
   * @nullable
   */
  options?: NotificationsOption[] | null;
  /** Its value is never returned */
  secret: boolean;
}

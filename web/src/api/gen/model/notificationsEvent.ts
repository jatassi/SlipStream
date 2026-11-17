import type { NotificationsEventGroup } from "./notificationsEventGroup.ts";

export interface NotificationsEvent {
  bellByDefault: boolean;
  /** Never chosen: a Test is sent only when the Admin asks */
  fixed: boolean;
  group: NotificationsEventGroup;
  id: string;
  label: string;
  /** A new connection gets it unless the Admin picks otherwise */
  onByDefault: boolean;
}

import type { NotificationsEvent } from "./notificationsEvent.ts";

export interface EventsOutputBody {
  /** Grouped by group, in the order the sheet shows them */
  events: NotificationsEvent[];
}

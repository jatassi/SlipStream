import type { LiveEventNotificationsSeenEvent } from "./liveEventNotificationsSeenEvent.ts";
import type { NotificationsSeen } from "./notificationsSeen.ts";

export interface LiveEventNotificationsSeen {
  event: LiveEventNotificationsSeenEvent;
  data: NotificationsSeen;
}

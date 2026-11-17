import type { LiveEventNotificationsChangedEvent } from "./liveEventNotificationsChangedEvent.ts";
import type { NotificationsChanged } from "./notificationsChanged.ts";

export interface LiveEventNotificationsChanged {
  event: LiveEventNotificationsChangedEvent;
  data: NotificationsChanged;
}

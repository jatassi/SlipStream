import {
  getGetNotificationBellQueryKey,
  getGetNotificationSettingsQueryKey,
  getListNotificationConnectionsQueryKey,
} from "../api/gen/query/notifications/notifications";
import { unreadKey } from "../notifications/feed-keys";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyFeed = everyPath(/^\/api\/v1\/notifications\/feed$/u);
const everyConnection = everyPath(/^\/api\/v1\/notifications\/connections\/[^/]+$/u);

/** The bell's feed and count, and the Notifications section's connections and settings. */
export const notificationEffects = {
  "notifications.changed": ({ feed, connections }, { refresh }) => {
    if (feed) {
      refresh(everyFeed);
    }
    if (connections) {
      refresh({ queryKey: getListNotificationConnectionsQueryKey() });
      refresh(everyConnection);
      refresh({ queryKey: getGetNotificationSettingsQueryKey() });
      refresh({ queryKey: getGetNotificationBellQueryKey() });
    }
  },
  "notifications.seen": (_data, { refresh }) => {
    refresh({ queryKey: unreadKey });
  },
} satisfies Pick<LiveEffects, "notifications.changed" | "notifications.seen">;

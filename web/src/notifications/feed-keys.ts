import { getListNotificationsQueryKey } from "../api/gen/query/notifications/notifications";

/** The bell's unread count reads the newest row alone. */
export const unreadParams = { limit: 1 } as const;

export const unreadKey = getListNotificationsQueryKey(unreadParams);

/** The feed sheet's pages, under the feed's path so a feed change refreshes them too. */
export const pagesKey = [...getListNotificationsQueryKey(), "pages"] as const;

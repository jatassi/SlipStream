import { infiniteQueryOptions } from "@tanstack/react-query";

import {
  getListNotificationsQueryKey,
  listNotifications,
} from "../api/gen/query/notifications/notifications";

// Both keys extend the feed's path, so notifications.changed refreshes them.

/** The bell's unread count reads the newest row alone. */
export const unreadParams = { limit: 1 } as const;

export const unreadKey = getListNotificationsQueryKey(unreadParams);

const pageSize = 30;

/** The feed sheet's rows, newest first, a page at a time. */
export const feedQuery = infiniteQueryOptions({
  queryKey: [...getListNotificationsQueryKey(), "pages"],
  queryFn: ({ pageParam, signal }) =>
    listNotifications(
      pageParam === null ? { limit: pageSize } : { limit: pageSize, before: pageParam },
      { signal },
    ),
  initialPageParam: null as number | null,
  getNextPageParam: (page) => page.before,
});

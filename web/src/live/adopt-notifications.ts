import type { QueryClient } from "@tanstack/react-query";

import type {
  ConnectionsOutputBody,
  NotificationsBell,
  NotificationsConnection,
  SettingsOutputBody,
} from "../api/gen/model";
import {
  getGetNotificationBellQueryKey,
  getGetNotificationConnectionQueryKey,
  getGetNotificationSettingsQueryKey,
  getListNotificationConnectionsQueryKey,
} from "../api/gen/query/notifications/notifications";

type Connections = ConnectionsOutputBody;

const byName = (a: NotificationsConnection, b: NotificationsConnection) =>
  a.name.localeCompare(b.name);

// A save answers with what it wrote, so the section shows it at once rather
// than after notifications.changed refetches it.

export function adoptConnection(queryClient: QueryClient, saved: NotificationsConnection) {
  queryClient.setQueryData(getGetNotificationConnectionQueryKey(saved.id), saved);
  queryClient.setQueryData<Connections>(
    getListNotificationConnectionsQueryKey(),
    (list) =>
      list && {
        connections: [...list.connections.filter((shown) => shown.id !== saved.id), saved].toSorted(
          byName,
        ),
      },
  );
}

export function dropConnection(queryClient: QueryClient, id: string) {
  queryClient.setQueryData<Connections>(
    getListNotificationConnectionsQueryKey(),
    (list) => list && { connections: list.connections.filter((shown) => shown.id !== id) },
  );
  queryClient.removeQueries({ queryKey: getGetNotificationConnectionQueryKey(id) });
}

export function adoptBell(queryClient: QueryClient, saved: NotificationsBell) {
  queryClient.setQueryData(getGetNotificationBellQueryKey(), saved);
}

export function adoptNotificationSettings(queryClient: QueryClient, saved: SettingsOutputBody) {
  queryClient.setQueryData(getGetNotificationSettingsQueryKey(), saved);
}

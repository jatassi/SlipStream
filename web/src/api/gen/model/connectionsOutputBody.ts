import type { NotificationsConnection } from "./notificationsConnection.ts";

export interface ConnectionsOutputBody {
  /** By name */
  connections: NotificationsConnection[];
}

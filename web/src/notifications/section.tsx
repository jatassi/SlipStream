import { getListNodesSuspenseQueryOptions } from "../api/gen/query/nodes/nodes";
import {
  getGetNotificationBellSuspenseQueryOptions,
  getGetNotificationSettingsSuspenseQueryOptions,
  getListNotificationConnectionsSuspenseQueryOptions,
  getListNotificationEventsSuspenseQueryOptions,
  getListNotificationProvidersSuspenseQueryOptions,
} from "../api/gen/query/notifications/notifications";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { BellSection } from "./bell-section";
import { ConnectionsSection } from "./connections-section";
import { NotificationSettingsPanel } from "./settings-panel";

function NotificationsSection() {
  return (
    <>
      <ConnectionsSection />
      <BellSection />
      <NotificationSettingsPanel />
    </>
  );
}

export const notificationsSection: SettingsSection = {
  id: "notifications",
  label: "Notifications",
  load: async (queryClient) => {
    await Promise.all([
      freshQuery(queryClient, getListNotificationConnectionsSuspenseQueryOptions()),
      freshQuery(queryClient, getGetNotificationSettingsSuspenseQueryOptions()),
      freshQuery(queryClient, getGetNotificationBellSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNotificationProvidersSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNotificationEventsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ]);
  },
  component: NotificationsSection,
  health: ["notifications"],
};

import { useQueryClient } from "@tanstack/react-query";

import type { SettingField as Field } from "../api/gen/model";
import {
  updateNotificationSettings,
  useGetNotificationSettingsSuspense,
} from "../api/gen/query/notifications/notifications";
import { adoptNotificationSettings } from "../live/adopt-notifications";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

export const settingKeys = {
  healthWait: "notifications.healthWait",
  giveUpAfter: "notifications.giveUpAfter",
  warnAfter: "notifications.warnAfter",
  keepFor: "notifications.keepFor",
} as const;

export function settingOf(settings: readonly Field[], key: string): Field {
  const field = settings.find((each) => each.key === key);
  if (field === undefined) {
    throw new Error(`the Notifications settings have no ${key}`);
  }
  return field;
}

/** The Notifications settings named by keys, each saved on its own. */
export function NotificationSettings({ keys }: { keys: readonly string[] }) {
  const queryClient = useQueryClient();
  const { settings } = useGetNotificationSettingsSuspense().data;
  return (
    <SettingGroup>
      {keys.map((key) => {
        const field = settingOf(settings, key);
        return (
          <SettingField
            key={key}
            field={field}
            onSave={async (value) => {
              adoptNotificationSettings(
                queryClient,
                await updateNotificationSettings({ values: { [key]: value } }),
              );
            }}
          />
        );
      })}
    </SettingGroup>
  );
}

/** The Health and Delivery settings. The Bell's own setting sits with the bell. */
export function NotificationSettingsPanel() {
  return (
    <>
      <Section title="Health">
        <NotificationSettings keys={[settingKeys.healthWait]} />
      </Section>
      <Section title="Delivery">
        <NotificationSettings keys={[settingKeys.giveUpAfter, settingKeys.warnAfter]} />
      </Section>
    </>
  );
}

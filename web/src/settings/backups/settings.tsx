import { useQueryClient } from "@tanstack/react-query";

import type { SettingField as Field } from "../../api/gen/model";
import { updateBackupSettings } from "../../api/gen/query/backups/backups";
import { adoptBackupSettings } from "../../live/adopt-backups";
import { SettingField, SettingGroup } from "../../ui/setting-field";

export const backupSettingKeys = {
  every: "tasks.backup.cadence",
  keepScheduled: "backups.keepScheduled",
  keepEach: "backups.keepEach",
  warnAfter: "backups.warnAfter",
  copyWaitLimit: "backups.copyWaitLimit",
  checkCopies: "backups.checkCopies",
} as const;

export const detailKeys = [
  backupSettingKeys.keepEach,
  backupSettingKeys.warnAfter,
  backupSettingKeys.copyWaitLimit,
  backupSettingKeys.checkCopies,
];

export function settingOf(settings: readonly Field[], key: string): Field {
  const field = settings.find((each) => each.key === key);
  if (field === undefined) {
    throw new Error(`the Backups settings have no ${key}`);
  }
  return field;
}

export const secondsOf = (field: Field): number =>
  typeof field.value === "number" ? field.value : 0;

export function BackupSettings({
  keys,
  settings,
}: {
  keys: readonly string[];
  settings: readonly Field[];
}) {
  const queryClient = useQueryClient();
  return (
    <SettingGroup>
      {keys.map((key) => (
        <SettingField
          key={key}
          field={settingOf(settings, key)}
          onSave={async (value) => {
            adoptBackupSettings(
              queryClient,
              await updateBackupSettings({ values: { [key]: value } }),
            );
          }}
        />
      ))}
    </SettingGroup>
  );
}

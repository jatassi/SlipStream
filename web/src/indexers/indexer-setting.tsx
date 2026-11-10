import { useQueryClient } from "@tanstack/react-query";

import {
  updateIndexerSettings,
  useGetIndexerSettingsSuspense,
} from "../api/gen/query/indexers/indexers";
import { adoptIndexerSettings } from "../live/adopt";
import { SettingField } from "../ui/setting-field";

/** The Indexer setting stored under key, saved through PATCH on its own. */
export function IndexerSetting({ settingKey }: { settingKey: string }) {
  const { settings } = useGetIndexerSettingsSuspense().data;
  const queryClient = useQueryClient();
  return settings
    .filter((field) => field.key === settingKey)
    .map((field) => (
      <SettingField
        key={field.key}
        field={field}
        onSave={async (value) => {
          adoptIndexerSettings(
            queryClient,
            await updateIndexerSettings({ values: { [field.key]: value } }),
          );
        }}
      />
    ));
}

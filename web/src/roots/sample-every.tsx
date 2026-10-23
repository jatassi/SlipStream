import { useQueryClient } from "@tanstack/react-query";

import { updateRootSettings, useGetRootSettingsSuspense } from "../api/gen/query/roots/roots";
import { adoptRootSettings } from "../live/adopt";
import { SettingField } from "../ui/setting-field";

/** "Check storage every": one stored value that Library roots and Health both edit. */
export function SampleEvery() {
  const { sampleEvery } = useGetRootSettingsSuspense().data;
  const queryClient = useQueryClient();
  const save = async (value: number | null) => {
    adoptRootSettings(queryClient, await updateRootSettings({ sampleEvery: value }));
  };
  return <SettingField field={sampleEvery} onSave={save} />;
}

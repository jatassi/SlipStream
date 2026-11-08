import { useQueryClient } from "@tanstack/react-query";

import type { DownloadclientsExternalClient as External } from "../api/gen/model";
import { updateDownloadClient } from "../api/gen/query/downloadclients/downloadclients";
import { adoptExternalClient } from "../live/adopt";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

type Timing = "checkEvery" | "importDelay";

/** A saved client's Check every and Import delay, each saved on its own. */
export function Timings({ client }: { client: External }) {
  const queryClient = useQueryClient();
  const save = async (key: Timing, value: number | null) => {
    const got = await updateDownloadClient(client.id, { [key]: value });
    adoptExternalClient(queryClient, got);
    toast(`Saved ${got.name}`);
  };
  return (
    <SettingGroup>
      <SettingField field={client.checkEvery} onSave={(value) => save("checkEvery", value)} />
      <SettingField field={client.importDelay} onSave={(value) => save("importDelay", value)} />
    </SettingGroup>
  );
}

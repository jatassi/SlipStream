import { useQueryClient } from "@tanstack/react-query";

import type { DownloadclientsSettings } from "../api/gen/model";
import {
  setDownloadClientSettings,
  useGetDownloadClientSettingsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import { adoptClientSettings } from "../live/adopt";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

type Key = keyof DownloadclientsSettings;

const groups: readonly { readonly title: string; readonly keys: readonly Key[] }[] = [
  { title: "Speed", keys: ["downloadLimit", "uploadLimit"] },
  { title: "Queue", keys: ["activeDownloads", "activeChecks", "stalledAfter"] },
  { title: "Peers", keys: ["connections", "utp", "dht"] },
];

/** A Node's built-in client settings kept on the Controller, each saved on its own and applied live. */
export function ClientLimits({ nodeId }: { nodeId: string }) {
  const settings = useGetDownloadClientSettingsSuspense(nodeId).data;
  const queryClient = useQueryClient();
  const save = async (key: Key, value: number | boolean | null) => {
    adoptClientSettings(
      queryClient,
      nodeId,
      await setDownloadClientSettings(nodeId, { [key]: value }),
    );
    toast("Torrent client settings saved");
  };
  return groups.map(({ title, keys }) => (
    <Section key={title} title={title}>
      <SettingGroup>
        {keys.map((key) => (
          <SettingField key={key} field={settings[key]} onSave={(value) => save(key, value)} />
        ))}
      </SettingGroup>
    </Section>
  ));
}

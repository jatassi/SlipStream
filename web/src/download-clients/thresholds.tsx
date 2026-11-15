import { useQueryClient } from "@tanstack/react-query";

import type { DownloadclientsThresholds } from "../api/gen/model";
import { setDownloadClientThresholds } from "../api/gen/query/downloadclients/downloadclients";
import { adoptThresholds } from "../live/adopt";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

type Key = keyof DownloadclientsThresholds;

const sections: readonly {
  readonly title: string;
  readonly about: string;
  readonly keys: readonly Key[];
}[] = [
  {
    title: "When a release counts as bad",
    about:
      "SlipStream treats a release as bad when its download gets no data for the first time below, or when its tracker reports it unregistered twice at least the second time below apart. SlipStream then adds the release to the Blocklist, removes its torrent and fails the Grab. The next automatic search picks another release.",
    keys: ["faultStall", "unregisteredGap"],
  },
  {
    title: "Magnet links",
    about: "A Grab of a magnet link fails when no peer sends its metadata in this long.",
    keys: ["metadataWait"],
  },
  {
    title: "Client errors",
    about:
      "A Grab fails when its external Download Client reports an error on its torrent for this long. The torrent stays in the client, and search can try another Release.",
    keys: ["clientErrorFail"],
  },
];

export function Thresholds({ thresholds }: { thresholds: DownloadclientsThresholds }) {
  const queryClient = useQueryClient();
  const save = async (key: Key, value: number | boolean | null) => {
    adoptThresholds(queryClient, await setDownloadClientThresholds({ [key]: value }));
    toast("Download Clients settings saved");
  };
  return sections.map(({ title, about, keys }) => (
    <Section key={title} title={title}>
      <p className="mb-3 text-ink-3">{about}</p>
      <SettingGroup>
        {keys.map((key) => (
          <SettingField key={key} field={thresholds[key]} onSave={(value) => save(key, value)} />
        ))}
      </SettingGroup>
    </Section>
  ));
}

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
    title: "When a Release counts as bad",
    about:
      "A download that gets no data at all for this long, or that its announce URL reports as unregistered twice this far apart, counts as a bad Release. Later automatic searches act on it.",
    keys: ["faultStall", "unregisteredGap"],
  },
  {
    title: "Magnet links",
    about: "A Grab of a magnet link fails when no peer sends its metadata in this long.",
    keys: ["metadataWait"],
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

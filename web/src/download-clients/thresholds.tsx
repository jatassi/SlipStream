import { useQueryClient } from "@tanstack/react-query";

import type { DownloadclientsThresholds } from "../api/gen/model";
import { setDownloadClientThresholds } from "../api/gen/query/downloadclients/downloadclients";
import { adoptThresholds } from "../live/adopt";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";

const thresholdKeys = [
  "faultStall",
  "unregisteredGap",
] as const satisfies readonly (keyof DownloadclientsThresholds)[];

export function Thresholds({ thresholds }: { thresholds: DownloadclientsThresholds }) {
  const queryClient = useQueryClient();
  return (
    <Section title="When a release counts as bad">
      <p className="mb-3 text-ink-3">
        A download that gets no data at all for this long, or that its tracker calls unregistered
        twice this far apart, counts as a bad release. Later automatic searches act on it.
      </p>
      <SettingGroup>
        {thresholdKeys.map((key) => (
          <SettingField
            key={key}
            field={thresholds[key]}
            onSave={async (value) => {
              const saved = await setDownloadClientThresholds({ [key]: value });
              adoptThresholds(queryClient, saved);
              toast("Download Clients settings saved");
            }}
          />
        ))}
      </SettingGroup>
    </Section>
  );
}

import { useQueryClient } from "@tanstack/react-query";

import {
  updateIndexerSettings,
  useGetIndexerSettingsSuspense,
} from "../api/gen/query/indexers/indexers";
import { adoptIndexerSettings } from "../live/adopt";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { TaskCadence } from "../tasks/task-cadence";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

/** indexers.longBackoff, how long an Indexer may fail before Health warns, and how often the health check tests each Indexer. */
export function LongBackoff() {
  const { settings } = useGetIndexerSettingsSuspense().data;
  const queryClient = useQueryClient();
  return (
    <Section title="Warnings">
      <div className="flex flex-col gap-3">
        <p className="text-ink-2">
          SlipStream retries a failing Indexer automatically, waiting longer after each failure.
          Health warns once every request to it has failed for this long.
        </p>
        <SettingGroup>
          {settings.map((field) => (
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
          ))}
        </SettingGroup>
        <p className="text-ink-2">
          The Indexer health check tests each Indexer that is on from the Node it searches from.
        </p>
        <KeyValues
          rows={[["Indexer health check", <TaskCadence key="check" task="indexer-health-check" />]]}
        />
      </div>
    </Section>
  );
}

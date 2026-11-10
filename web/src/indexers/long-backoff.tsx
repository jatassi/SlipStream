import { KeyValues } from "../routes/_app/nodes/-facts";
import { TaskCadence } from "../tasks/task-cadence";
import { Section } from "../ui/section";
import { SettingGroup } from "../ui/setting-field";
import { IndexerSetting } from "./indexer-setting";

/** indexers.longBackoff, how long an Indexer may fail before Health warns, and how often the health check tests each Indexer. */
export function LongBackoff() {
  return (
    <Section title="Warnings">
      <div className="flex flex-col gap-3">
        <p className="text-ink-2">
          SlipStream retries a failing Indexer automatically, waiting longer after each failure.
          Health warns once every request to it has failed for this long.
        </p>
        <SettingGroup>
          <IndexerSetting settingKey="indexers.longBackoff" />
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

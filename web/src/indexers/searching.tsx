import { Section } from "../ui/section";
import { SettingGroup } from "../ui/setting-field";
import { IndexerSetting } from "./indexer-setting";

/** indexers.searchWait, how long a search waits for each Indexer. */
export function Searching() {
  return (
    <Section title="Searching">
      <div className="flex flex-col gap-3">
        <p className="text-ink-2">
          SlipStream waits this long for each Indexer to respond to a search. A wait shorter than a
          Challenge solver&apos;s timeout ends a solve before it finishes.
        </p>
        <SettingGroup>
          <IndexerSetting settingKey="indexers.searchWait" />
        </SettingGroup>
      </div>
    </Section>
  );
}

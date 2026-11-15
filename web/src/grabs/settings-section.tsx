import { useQueryClient } from "@tanstack/react-query";

import type { SettingField as Field } from "../api/gen/model";
import {
  getGetSearchSettingsSuspenseQueryOptions,
  updateSearchSettings,
  useGetSearchSettingsSuspense,
} from "../api/gen/query/grabs/grabs";
import { freshQuery } from "../api/query-client";
import { adoptSearchSettings } from "../live/adopt-search-settings";
import type { SettingsSection } from "../settings/section";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const keys = {
  backlogOn: "search.backlogOn",
  backlogUpgrades: "search.backlogUpgrades",
  rssOn: "search.rssOn",
  longestGap: "search.longestGap",
  fullPace: "search.fullPace",
  stopCount: "search.stopCount",
  stopWindow: "search.stopWindow",
  wait: "indexers.searchWait",
} as const;

const limits = "Searches never go past an Indexer's limits.";

/** The section's lead, from whether Backlog search and RSS sync are on. */
export function leadSentence(backlogOn: boolean, rssOn: boolean): string {
  if (backlogOn && rssOn) {
    return `SlipStream searches for what's wanted and reads new uploads on a schedule. ${limits}`;
  }
  if (rssOn) {
    return `SlipStream reads new uploads on a schedule and searches only when you ask. ${limits}`;
  }
  if (backlogOn) {
    return `SlipStream searches for what's wanted on a schedule. ${limits}`;
  }
  return `SlipStream searches only when you ask. ${limits}`;
}

function fieldOf(settings: readonly Field[], key: string): Field {
  const field = settings.find((each) => each.key === key);
  if (field === undefined) {
    throw new Error(`the Automatic search settings have no ${key}`);
  }
  return field;
}

type Save = (key: string) => (value: unknown) => Promise<void>;

function useSave(): Save {
  const queryClient = useQueryClient();
  return (key) => async (value) => {
    adoptSearchSettings(queryClient, await updateSearchSettings({ values: { [key]: value } }));
  };
}

function switchOn(field: Field): boolean {
  if (field.kind !== "switch") {
    throw new Error(`${field.key} is no switch`);
  }
  return field.value;
}

const noteLook = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

/** "Stop searching again after [3 failed Grabs] in [7 days]", each value saved on its own. */
function StopLine({ settings, save }: { settings: readonly Field[]; save: Save }) {
  return (
    <div className="flex flex-col gap-1">
      <p className="flex flex-wrap items-center gap-x-1.5 text-ink-1">
        Stop searching again after
        <SettingField
          field={fieldOf(settings, keys.stopCount)}
          onSave={save(keys.stopCount)}
          variant="row"
        />
        in
        <SettingField
          field={fieldOf(settings, keys.stopWindow)}
          onSave={save(keys.stopWindow)}
          variant="row"
        />
      </p>
      <p className={noteLook}>
        A cell stops searching again after this many failed Grabs. With 1, the first failed Grab
        stops it. Backlog search still searches it at its own pace.
      </p>
    </div>
  );
}

/** The machinery: the longest gap, full pace, the stop and the wait for each Indexer. */
function Machinery({ settings, save }: { settings: readonly Field[]; save: Save }) {
  const full = (key: string) => <SettingField field={fieldOf(settings, key)} onSave={save(key)} />;
  return (
    <Details fields>
      <div className="mt-2 flex w-full flex-col gap-4">
        <SettingGroup>
          {full(keys.longestGap)}
          {full(keys.fullPace)}
          <StopLine settings={settings} save={save} />
          <div className="flex flex-col gap-1">
            {full(keys.wait)}
            <p className={noteLook}>
              A wait shorter than a Challenge solver&apos;s timeout ends a solve before it finishes.
            </p>
          </div>
        </SettingGroup>
      </div>
    </Details>
  );
}

/** Backlog search's and RSS sync's switches first, the rest under Details. */
function AutomaticSearchSection() {
  const { settings } = useGetSearchSettingsSuspense().data;
  const save = useSave();
  const on = (key: string) => switchOn(fieldOf(settings, key));
  return (
    <Section title="Automatic search">
      <div className="flex flex-col gap-5">
        <p className="text-ink-2">{leadSentence(on(keys.backlogOn), on(keys.rssOn))}</p>
        <SettingGroup>
          {[keys.backlogOn, keys.backlogUpgrades, keys.rssOn].map((key) => (
            <SettingField key={key} field={fieldOf(settings, key)} onSave={save(key)} />
          ))}
        </SettingGroup>
        <Machinery settings={settings} save={save} />
      </div>
    </Section>
  );
}

export const automaticSearchSection: SettingsSection = {
  id: "automatic-search",
  label: "Automatic search",
  load: async (queryClient) => {
    await freshQuery(queryClient, getGetSearchSettingsSuspenseQueryOptions());
  },
  component: AutomaticSearchSection,
};

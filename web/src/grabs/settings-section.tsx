import { useQueryClient } from "@tanstack/react-query";

import type { SettingField as Field } from "../api/gen/model";
import {
  getGetSearchSettingsSuspenseQueryOptions,
  updateSearchSettings,
  useGetSearchSettingsSuspense,
} from "../api/gen/query/grabs/grabs";
import { freshQuery } from "../api/query-client";
import {
  adoptSearchSettings,
  backlogCadenceKey,
  rssCadenceKey,
} from "../live/adopt-search-settings";
import type { SettingsSection } from "../settings/section";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const keys = {
  backlogOn: "search.backlogOn",
  backlogUpgrades: "search.backlogUpgrades",
  backlogEvery: backlogCadenceKey,
  rssOn: "search.rssOn",
  rssEvery: rssCadenceKey,
  longestGap: "search.longestGap",
  fullPace: "search.fullPace",
  stopCount: "search.stopCount",
  stopWindow: "search.stopWindow",
  wait: "indexers.searchWait",
} as const;

const limits = "Searches never go past an Indexer's limits.";

const units = [
  [24 * 3600, "day"],
  [3600, "hour"],
  [60, "minute"],
] as const;

/** A cadence in words: "every 8 hours", "every 15 minutes", "every day". */
export function everyWords(interval: number): string {
  const [size, unit] = units.find(([each]) => interval % each === 0) ?? [1, "second"];
  const count = interval / size;
  return count === 1 ? `every ${unit}` : `every ${count} ${unit}s`;
}

type Schedule = { on: boolean; every: number };

/** The section's lead, from whether Backlog search, its upgrades and RSS sync are on and how often each runs. */
export function leadSentence(backlog: Schedule & { upgrades: boolean }, rss: Schedule): string {
  const what = backlog.upgrades ? "missing files and upgrades" : "missing files";
  const searches = `searches for ${what} ${everyWords(backlog.every)}`;
  const reads = `reads new uploads ${everyWords(rss.every)}`;
  if (backlog.on && rss.on) {
    return `SlipStream ${searches} and ${reads}. ${limits}`;
  }
  if (rss.on) {
    return `SlipStream ${reads} and searches only when you ask. ${limits}`;
  }
  if (backlog.on) {
    return `SlipStream ${searches}. ${limits}`;
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
    await adoptSearchSettings(
      queryClient,
      key,
      await updateSearchSettings({ values: { [key]: value } }),
    );
  };
}

function switchOn(field: Field): boolean {
  if (field.kind !== "switch") {
    throw new Error(`${field.key} is no switch`);
  }
  return field.value;
}

function seconds(field: Field): number {
  if (field.kind !== "duration") {
    throw new Error(`${field.key} is no duration`);
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
        After a Grab fails, SlipStream searches for the file again right away. After this many
        failed Grabs, it stops, and only Backlog search looks for the file. With 1, SlipStream never
        searches again right away.
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
              Set it longer than the Challenge solver&apos;s timeout, or searches stop before the
              solver clears the challenge.
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
  const every = (key: string) => seconds(fieldOf(settings, key));
  return (
    <Section title="Automatic search">
      <div className="flex flex-col gap-5">
        <p className="text-ink-2">
          {leadSentence(
            {
              on: on(keys.backlogOn),
              upgrades: on(keys.backlogUpgrades),
              every: every(keys.backlogEvery),
            },
            { on: on(keys.rssOn), every: every(keys.rssEvery) },
          )}
        </p>
        <SettingGroup>
          {[keys.backlogOn, keys.backlogUpgrades, keys.backlogEvery, keys.rssOn, keys.rssEvery].map(
            (key) => (
              <SettingField key={key} field={fieldOf(settings, key)} onSave={save(key)} />
            ),
          )}
        </SettingGroup>
        <Machinery settings={settings} save={save} />
      </div>
    </Section>
  );
}

/** RSS sync's cadence as the Automatic search section stores it, for where the task isn't listed. */
export function RSSEveryField() {
  const { settings } = useGetSearchSettingsSuspense().data;
  const save = useSave();
  return (
    <SettingField
      field={fieldOf(settings, keys.rssEvery)}
      onSave={save(keys.rssEvery)}
      variant="row"
    />
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

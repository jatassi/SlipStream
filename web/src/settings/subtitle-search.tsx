import type { SettingField as Field } from "../api/gen/model";
import { cadenceKey, useSubtitleWrites } from "../subtitles/settings-api";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const shown = ["subtitles.searchOn", "subtitles.searchUpgrades", cadenceKey];

const underDetails = [
  "subtitles.longestGap",
  "subtitles.fullPace",
  "subtitles.triesPerSearch",
  "subtitles.sync.match",
  "subtitles.sync.maxOffset",
  "subtitles.sync.framerates",
  "subtitles.providerWarnAfter",
];

const meanings: Readonly<Record<string, string>> = {
  "subtitles.sync.match":
    "0% is what random timing scores, and 100% means every line lands on speech.",
  none: "None: shifts only, never stretches.",
  standard: "Standard rates: common conversions, such as 25 to 23.976 fps.",
  range:
    "Standard rates and any between 0.9 and 1.1 times: also finds unusual conversions, and takes longer.",
};

const meaningOf = (field: Field): string | undefined =>
  meanings[field.kind === "choice" ? field.value : field.key];

function Setting({ field }: { field: Field }) {
  const { saveSetting } = useSubtitleWrites();
  const meaning = meaningOf(field);
  return (
    <div className="flex flex-col gap-1">
      <SettingField field={field} onSave={(value) => saveSetting(field.key, value)} />
      {meaning !== undefined && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">{meaning}</p>
      )}
    </div>
  );
}

/** Subtitle search's switches and interval, with the machinery under Details. */
export function SubtitleSearch({ settings }: { settings: readonly Field[] }) {
  const fields = (keys: readonly string[]) =>
    settings
      .filter((field) => keys.includes(field.key))
      .map((field) => <Setting key={field.key} field={field} />);
  return (
    <Section title="Subtitle search">
      <div className="flex flex-col gap-5">
        <SettingGroup>{fields(shown)}</SettingGroup>
        <Details fields>
          <div className="mt-2 flex w-full flex-col gap-4">
            <SettingGroup>{fields(underDetails)}</SettingGroup>
          </div>
        </Details>
      </div>
    </Section>
  );
}

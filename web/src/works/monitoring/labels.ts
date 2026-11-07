import type { WorksContentType, WorksItem, WorksMonitoringChangePreset } from "../../api/gen/model";
import { type Noun, countWords } from "../../format";

export type Preset = WorksMonitoringChangePreset;

export const lowered = ({ one, many }: Noun): Noun => ({
  one: one.toLowerCase(),
  many: many.toLowerCase(),
});

/** The outermost level, which "First" and "Latest" count by. */
function outerLevel(type: WorksContentType): Noun {
  const [level] = type.levels;
  if (level === undefined) {
    throw new Error(`${type.id} has many Items and no level`);
  }
  return lowered(level);
}

/** Each preset's label, in the order the choice lists them; a new preset fails the build here. */
export const presetLabels: { readonly [P in Preset]: (type: WorksContentType) => string } = {
  all: () => "All",
  future: () => "Future",
  missing: () => "Missing",
  existing: () => "Existing",
  first: (type) => `First ${outerLevel(type).one}`,
  latest: (type) => `Latest ${outerLevel(type).one}`,
  none: () => "None",
};

export const isPreset = (key: string): key is Preset => Object.hasOwn(presetLabels, key);

export const presets: readonly Preset[] = Object.keys(presetLabels).filter((key) => isPreset(key));

/** The type's verb for an Item coming out, such as air. */
export const releaseVerb = (type: Pick<WorksContentType, "releases">): string =>
  type.releases === "" ? "come out" : type.releases;

export const includeLabel = (aside: Noun): string => `Include ${lowered(aside).many}`;

export const monitorNewLabel = (type: WorksContentType): string =>
  `Monitor new ${lowered(type.item).many}`;

/** "Monitoring 24 of 62 stories", counting the Items the source still lists. */
export function monitoringWords(type: WorksContentType, items: readonly WorksItem[]): string {
  const live = items.filter(({ withdrawnAt }) => withdrawnAt === null);
  const monitored = live.filter(({ monitored: on }) => on).length;
  return `Monitoring ${monitored} of ${countWords(live.length, lowered(type.item))}`;
}

import { HttpResponse, type RequestHandler, http } from "msw";

import type { SettingField } from "../api/gen/model";
import { SearchSettingsInputBody } from "../api/gen/zod/slipStream.schemas.zod";

const hour = 3600;
const day = 24 * hour;

const about = {
  source: "database",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
} as const;

const onOff = (key: string, label: string): SettingField => ({
  ...about,
  kind: "switch",
  key,
  label,
  value: true,
  running: true,
  default: true,
});

const duration = (
  key: string,
  label: string,
  value: number,
  [min, max, step]: readonly [number, number, number],
): SettingField => ({
  ...about,
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  step,
});

/** The Automatic search settings at their defaults, in the order the server lists them. */
export const searchSettings = (): SettingField[] => [
  onOff("search.backlogOn", "Backlog search"),
  onOff("search.backlogUpgrades", "Backlog search looks for upgrades"),
  onOff("search.rssOn", "RSS sync"),
  duration(
    "search.longestGap",
    "Search a Want that keeps coming up empty at least every",
    7 * day,
    [day, 90 * day, hour],
  ),
  duration("search.fullPace", "Don't space out searches for Items released in the last", 14 * day, [
    0,
    90 * day,
    day,
  ]),
  {
    ...about,
    kind: "count",
    key: "search.stopCount",
    label: "Stop searching again after",
    value: 3,
    running: 3,
    default: 3,
    min: 1,
    max: 20,
    unit: { one: "failed Grab", many: "failed Grabs" },
  },
  duration("search.stopWindow", "Count failed Grabs in the last", 7 * day, [hour, 30 * day, hour]),
  duration("indexers.searchWait", "Wait for each Indexer up to", 120, [10, 600, 1]),
];

export type SearchSettingsWorld = {
  settings: SettingField[];
  writes: SearchSettingsInputBody[];
  handlers: RequestHandler[];
};

/** Answers GET and PATCH /search-settings from one list, keeping each write. */
export function searchSettingsWorld(): SearchSettingsWorld {
  const world: SearchSettingsWorld = {
    settings: searchSettings(),
    writes: [],
    handlers: [],
  };
  world.handlers = [
    http.get("*/api/v1/search-settings", () => HttpResponse.json({ settings: world.settings })),
    http.patch("*/api/v1/search-settings", async ({ request }) => {
      const body = SearchSettingsInputBody.parse(await request.json());
      world.writes.push(body);
      const next = world.settings.map((field) => {
        const sent = body.values[field.key];
        if (sent === undefined) {
          return field;
        }
        const value: unknown = sent ?? field.default;
        return Object.assign(structuredClone(field), {
          value,
          running: value,
          isDefault: value === field.default,
        });
      });
      world.settings = next;
      return HttpResponse.json({ settings: world.settings });
    }),
  ];
  return world;
}

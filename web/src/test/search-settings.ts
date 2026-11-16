import { HttpResponse, type RequestHandler, http } from "msw";

import type { SettingField } from "../api/gen/model";
import { SearchSettingsInputBody } from "../api/gen/zod/slipStream.schemas.zod";
import { refused } from "./roots";

const minute = 60;
const hour = 60 * minute;
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
  extra: { floor?: string } = {},
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
  ...extra,
});

/** The Automatic search settings at their defaults, in the order the server lists them. */
export const searchSettings = (): SettingField[] => [
  onOff("search.backlogOn", "Backlog search"),
  onOff("search.backlogUpgrades", "Backlog search looks for upgrades"),
  duration(
    "tasks.backlog-search.cadence",
    "Backlog search runs every",
    8 * hour,
    [hour, 7 * day, hour],
    {
      floor: "Each run can send a search to every Indexer's site.",
    },
  ),
  onOff("search.rssOn", "RSS sync"),
  duration(
    "tasks.rss-sync.cadence",
    "RSS sync runs every",
    15 * minute,
    [10 * minute, 2 * hour, minute],
    {
      floor: "Many trackers ban clients that read their RSS feed more often than every 10 minutes.",
    },
  ),
  duration(
    "search.longestGap",
    "Search for a file that keeps coming up empty at least every",
    7 * day,
    [day, 90 * day, hour],
  ),
  duration(
    "search.fullPace",
    "Don't space out searches for anything released in the last",
    14 * day,
    [0, 90 * day, day],
  ),
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

const gapKey = "search.longestGap";
const everyKey = "tasks.backlog-search.cadence";

function spelled(seconds: number): string {
  const [n, unit] = seconds % day === 0 ? [seconds / day, "day"] : [seconds / hour, "hour"];
  return n === 1 ? `1 ${unit}` : `${n} ${unit}s`;
}

/** The server's refusal of a longest gap shorter than Backlog search's interval, on the key that moved. */
function gapRefusal(settings: readonly SettingField[], values: Record<string, unknown>) {
  const next = (key: string): number => {
    const sent = values[key];
    const field = settings.find((each) => each.key === key);
    return Number(sent ?? (key in values ? field?.default : field?.value));
  };
  const moved = [gapKey, everyKey].find((key) => key in values);
  if (moved === undefined || next(gapKey) >= next(everyKey)) {
    return null;
  }
  const words =
    moved === gapKey
      ? `Use ${spelled(next(everyKey))} or longer. Backlog search runs every ${spelled(next(everyKey))}.`
      : `Use ${spelled(next(gapKey))} or shorter. A file that keeps coming up empty is searched for at least every ${spelled(next(gapKey))}.`;
  return refused(
    422,
    "longest_gap_too_short",
    "Backlog search runs less often than a file that keeps coming up empty is searched for",
    {
      detail: words,
      errors: [{ location: `body.values.${moved}`, message: words }],
    },
  );
}

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
      const refusal = gapRefusal(world.settings, body.values);
      if (refusal !== null) {
        return refusal;
      }
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

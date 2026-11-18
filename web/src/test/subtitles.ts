import type {
  NodesNode,
  SettingField,
  SubtitlesLead,
  SubtitlesProfileLanguages,
  SubtitlesProvider,
  WorksContentType,
} from "../api/gen/model";
import { cinema } from "./library";
import { home, homeId } from "./nodes";

const hour = 3600;
export const day = 24 * hour;

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

const count = (
  key: string,
  label: string,
  value: number,
  [min, max]: readonly [number, number],
  unit: { one: string; many: string },
): SettingField => ({
  ...about,
  kind: "count",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  unit,
});

const framerates: SettingField = {
  ...about,
  kind: "choice",
  key: "subtitles.sync.framerates",
  label: "Framerate changes to try",
  value: "standard",
  running: "standard",
  default: "standard",
  options: [
    { value: "none", label: "None" },
    { value: "standard", label: "Standard rates" },
    { value: "range", label: "Standard rates and any between 0.9 and 1.1 times" },
  ],
};

/** The Subtitles section's settings at their defaults, in the order the server lists them. */
export const subtitleSettings = (): SettingField[] => [
  onOff("subtitles.searchOn", "Subtitle search"),
  onOff("subtitles.searchUpgrades", "Subtitle search looks for better subtitles"),
  duration("tasks.subtitle-search.cadence", "Subtitle search runs every", 8 * hour, [
    hour,
    7 * day,
    hour,
  ]),
  duration(
    "subtitles.longestGap",
    "Search for a subtitle that keeps coming up empty at least every",
    7 * day,
    [day, 90 * day, hour],
  ),
  duration(
    "subtitles.fullPace",
    "Don't space out searches for anything released in the last",
    14 * day,
    [0, 90 * day, day],
  ),
  count("subtitles.triesPerSearch", "Try at most this many subtitles per search", 3, [1, 10], {
    one: "subtitle",
    many: "subtitles",
  }),
  count(
    "subtitles.sync.match",
    "Keep a subtitle only when its timing match is at least",
    30,
    [10, 95],
    { one: "%", many: "%" },
  ),
  duration("subtitles.sync.maxOffset", "Largest timing shift to accept", 30, [5, 300, 1]),
  framerates,
  duration("subtitles.providerWarnAfter", "Warn when a Subtitle Provider keeps failing for", hour, [
    600,
    day,
    60,
  ]),
];

/** A Storage Node's audio scans at once, out of the default of 1. */
export const slotsField = (value: number): SettingField => ({
  ...about,
  kind: "count",
  key: "subtitles.mapSlots",
  label: "Audio scans at once on this Node",
  value,
  running: value,
  default: 1,
  isDefault: value === 1,
  min: 1,
  max: 8,
  unit: { one: "scan", many: "scans" },
});

export const movies: WorksContentType = { ...cinema, name: "Movies" };

export const tv: WorksContentType = {
  ...cinema,
  id: "tv",
  name: "TV",
  levels: [{ one: "Season", many: "Seasons" }],
};

const noState = { since: null, until: null, words: null } as const;

export const subdl: SubtitlesProvider = {
  id: "subdl",
  name: "SubDL",
  kinds: ["feature", "episode"],
  position: 1,
  on: true,
  needsKey: true,
  hasKey: false,
  keyPage: "https://subdl.invalid/panel/api",
  account: null,
  plan: null,
  allowance: null,
  state: { kind: "needs_key", ...noState },
};

export const subsource: SubtitlesProvider = {
  ...subdl,
  id: "subsource",
  name: "SubSource",
  position: 2,
  keyPage: "https://subsource.invalid/profile",
};

export const gestdown: SubtitlesProvider = {
  ...subdl,
  id: "gestdown",
  name: "Gestdown",
  kinds: ["episode"],
  position: 3,
  needsKey: false,
  keyPage: null,
  state: { kind: "ready", ...noState },
};

export const english = {
  language: "en",
  forced: false,
  hearingImpaired: "allow",
  audio: "always",
} as const;

export const moviesLanguages: SubtitlesProfileLanguages = {
  id: "0199e1f2-7a3b-7c4d-8e5f-000000000011",
  name: "Movies",
  contentType: "cinema",
  languages: [english],
};

export const readyLead: SubtitlesLead = {
  kind: "ready",
  uncovered: [],
  missing: 1204,
  interval: 8 * hour,
  scans: [{ nodeId: homeId, name: "home", left: 312 }],
};

export const storageHome: NodesNode = {
  ...home,
  roles: [...home.roles, { role: "storage", state: "holding" }],
};

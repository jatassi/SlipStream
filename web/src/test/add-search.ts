import type {
  SettingChoiceField,
  SettingSwitchField,
  WorksScopeCount,
  WorksSearchChoiceValue,
} from "../api/gen/model";
import { anthologies } from "./anthology";

/** What each add search scope searches of the Items a preset monitors. */
export const searching = (
  monitored: number,
  firstLevel: number,
  latestLevel: number,
  firstItem: number,
): WorksScopeCount[] => [
  { scope: "monitored", items: monitored },
  { scope: "first_level", items: firstLevel },
  { scope: "latest_level", items: latestLevel },
  { scope: "first_item", items: firstItem },
  { scope: "none", items: 0 },
];

/** Films' add.search switch, on by default. */
export const addSearch = (value: boolean): SettingSwitchField => ({
  key: "cinema.add.search",
  label: "Search after adding",
  kind: "switch",
  source: value ? "default" : "database",
  applies: "live",
  isDefault: value,
  pendingRestart: false,
  default: true,
  running: value,
  value,
});

/** Anthologies' add.search choice of its scopes, Everything monitored by default. */
export const anthologySearch = (value: WorksSearchChoiceValue): SettingChoiceField => ({
  key: "anthology.add.search",
  label: "Search after adding",
  kind: "choice",
  source: value === "monitored" ? "default" : "database",
  applies: "live",
  isDefault: value === "monitored",
  pendingRestart: false,
  default: "monitored",
  running: value,
  value,
  options: anthologies.addSearch.choices.map(({ value: each, label }) => ({ value: each, label })),
});

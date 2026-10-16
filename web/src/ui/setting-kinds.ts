import { countKind, durationKind } from "./setting-amounts";
import type { Kind, SettingKind } from "./setting-kind";
import { choiceKind, windowKind } from "./setting-picks";

// A new kind of setting is its member in Fields and one renderer here.
export const kinds: { readonly [K in SettingKind]: Kind<K> } = {
  duration: durationKind,
  count: countKind,
  choice: choiceKind,
  window: windowKind,
};

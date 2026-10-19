import { countKind, durationKind, portKind } from "./setting-amounts";
import type { Kind, SettingKind } from "./setting-kind";
import { choiceKind, switchKind, windowKind } from "./setting-picks";
import { spaceKind } from "./setting-space";

export const kinds: { readonly [K in SettingKind]: Kind<K> } = {
  duration: durationKind,
  count: countKind,
  choice: choiceKind,
  window: windowKind,
  port: portKind,
  switch: switchKind,
  space: spaceKind,
};

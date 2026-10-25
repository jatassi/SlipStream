import { folderKind } from "../folders/setting-folder";
import { addressKind } from "./setting-address";
import { countKind, durationKind, portKind } from "./setting-amounts";
import type { Kind, SettingKind } from "./setting-kind";
import { choiceKind, switchKind, windowKind } from "./setting-picks";
import { rateKind } from "./setting-rate";
import { spaceKind } from "./setting-space";

export const kinds: { readonly [K in SettingKind]: Kind<K> } = {
  duration: durationKind,
  count: countKind,
  choice: choiceKind,
  window: windowKind,
  port: portKind,
  switch: switchKind,
  rate: rateKind,
  address: addressKind,
  folder: folderKind,
  space: spaceKind,
};

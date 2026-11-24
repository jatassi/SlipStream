import { folderKind } from "../folders/setting-folder";
import { addressKind, listenKind } from "./setting-address";
import { countKind, durationKind } from "./setting-amounts";
import type { Kind, SettingKind } from "./setting-kind";
import { listKind } from "./setting-list";
import { choiceKind, switchKind, windowKind } from "./setting-picks";
import { rateKind } from "./setting-rate";
import { sizeKind } from "./setting-size";
import { spaceKind } from "./setting-space";
import { ipKind, templateKind } from "./setting-template";

export const kinds: { readonly [K in SettingKind]: Kind<K> } = {
  duration: durationKind,
  count: countKind,
  choice: choiceKind,
  window: windowKind,
  listen: listenKind,
  switch: switchKind,
  rate: rateKind,
  address: addressKind,
  folder: folderKind,
  space: spaceKind,
  template: templateKind,
  list: listKind,
  size: sizeKind,
  ip: ipKind,
};

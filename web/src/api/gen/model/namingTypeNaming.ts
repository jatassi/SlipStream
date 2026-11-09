import type { NamingProblem } from "./namingProblem.ts";
import type { SettingField } from "./settingField.ts";

export interface NamingTypeNaming {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** What the save check refuses in the stored settings once renaming is on; empty when every file of a Work shows as one item on Plex, Jellyfin and Emby */
  problems: NamingProblem[];
  /** The rename switch, the folder templates outermost first, the file templates, the colon mode and its text, and the multi-episode style when the type's file templates number its Items */
  settings: SettingField[];
}

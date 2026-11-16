import type { SettingField } from "./settingField.ts";

export interface WorksOptionsChanged {
  levelFolders: boolean;
  /** The change fetches the Work again, so its Items may move between levels */
  moves: boolean;
  /** The Content Type's own options for the Work */
  options: SettingField[];
  /** Items the source no longer lists under the new options; they keep their files */
  withdraws: number;
}

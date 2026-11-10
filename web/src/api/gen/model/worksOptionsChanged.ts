import type { SettingField } from "./settingField.ts";

export interface WorksOptionsChanged {
  levelFolders: boolean;
  /** The change fetches the Work again, so its Items may move between levels */
  moves: boolean;
  /** The Content Type's own options for the Work */
  options: SettingField[];
  /** Live Items the change withdraws: the source doesn't list them under the new options, and they keep their files */
  withdraws: number;
}

import type { SettingSwitchFieldApplies } from "./settingSwitchFieldApplies.ts";
import type { SettingSwitchFieldKind } from "./settingSwitchFieldKind.ts";
import type { SettingSwitchFieldSource } from "./settingSwitchFieldSource.ts";

export interface SettingSwitchField {
  applies: SettingSwitchFieldApplies;
  default: boolean;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingSwitchFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  running: boolean;
  source: SettingSwitchFieldSource;
  value: boolean;
}

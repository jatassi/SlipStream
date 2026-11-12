import type { SettingSizeFieldApplies } from "./settingSizeFieldApplies.ts";
import type { SettingSizeFieldKind } from "./settingSizeFieldKind.ts";
import type { SettingSizeFieldSource } from "./settingSizeFieldSource.ts";

export interface SettingSizeField {
  applies: SettingSizeFieldApplies;
  /** Bytes */
  default: number;
  /** Where the default comes from, when it is another setting's value */
  defaultFrom?: string;
  /** The setting has no override; Reset to default leaves it unchanged */
  isDefault: boolean;
  key: string;
  kind: SettingSizeFieldKind;
  label: string;
  /** The environment variable that sets the value; the control is read-only */
  lockedBy?: string;
  /** Bytes */
  max: number;
  /** Bytes */
  min: number;
  /** The saved value differs from the running one until the Node restarts */
  pendingRestart: boolean;
  /** Bytes */
  running: number;
  source: SettingSizeFieldSource;
  /** Bytes */
  value: number;
}

import type { SettingCountField } from "./settingCountField.ts";
import type { SettingDurationField } from "./settingDurationField.ts";

export interface IdentitySettings {
  addressFailures: SettingCountField;
  addressWindow: SettingDurationField;
  emailLinkLifetime: SettingDurationField;
  pinOffAfter: SettingCountField;
  pinWait: SettingDurationField;
  pinWaitAfter: SettingCountField;
  reconfirmWindow: SettingDurationField;
  resetLinkLifetime: SettingDurationField;
  sessionLifetime: SettingDurationField;
  signupLinkLifetime: SettingDurationField;
}

import type { MailSettingsSecurity } from "./mailSettingsSecurity.ts";

export interface MailSettings {
  fromAddress: string;
  fromName: string;
  /** Empty until email is set up */
  host: string;
  /** Whether a password is saved; it is never shown */
  passwordSet: boolean;
  /** @minimum 1 */
  port: number;
  security: MailSettingsSecurity;
  /** Empty when the server takes no sign-in */
  username: string;
}

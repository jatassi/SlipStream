import type { MailSettingsPatchSecurity } from "./mailSettingsPatchSecurity.ts";

export interface MailSettingsPatch {
  /** @maxLength 254 */
  fromAddress?: string;
  /** @maxLength 100 */
  fromName?: string;
  /** @maxLength 253 */
  host?: string;
  /**
   * Write-only; empty clears it
   * @maxLength 1024
   */
  password?: string;
  /** @minimum 1 */
  port?: number;
  security?: MailSettingsPatchSecurity;
  /** @maxLength 254 */
  username?: string;
}

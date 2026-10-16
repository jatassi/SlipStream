import { MailOutcome, MailSettings, MailSettingsPatch } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get the SMTP server every email goes through
 */
export const GetMailSettingsResponse = MailSettings;

/**
 * @summary Change the SMTP server every email goes through
 */
export const UpdateMailSettingsBody = MailSettingsPatch;

export const UpdateMailSettingsResponse = MailSettings;

/**
 * @summary Send a test email to your own address and answer the mail server's reply
 */
export const SendTestMailResponse = MailOutcome;

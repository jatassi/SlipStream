import * as zod from "zod";

import { HomeSettings, HomeSummary, SettingsInputBody } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get Home: the moving Grabs, what needs attention and the newest finished Imports
 */
export const GetHomeQueryParams = zod.object({
  since: zod.iso
    .datetime({ offset: true })
    .describe("The browser's local midnight, from which today's totals count"),
});

export const GetHomeResponse = HomeSummary;

/**
 * @summary Get Home's settings
 */
export const GetHomeSettingsResponse = HomeSettings;

/**
 * @summary Change Home's settings by key, all or none
 */
export const UpdateHomeSettingsBody = SettingsInputBody;

export const UpdateHomeSettingsResponse = HomeSettings;

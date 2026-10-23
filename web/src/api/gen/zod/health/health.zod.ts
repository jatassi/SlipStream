import * as zod from "zod";

import {
  HealthChecked,
  HealthReport,
  HealthSettings,
  HealthSettingsInputBody,
  HealthVolume,
  HealthVolumeInputBody,
  HealthVolumeList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get what is wrong with SlipStream, by category
 */
export const GetSystemHealthResponse = HealthReport;

/**
 * @summary Check every category now
 */
export const CheckSystemHealthResponse = HealthChecked;

/**
 * @summary Get the global Storage levels
 */
export const GetHealthSettingsResponse = HealthSettings;

/**
 * @summary Change the global Storage levels
 */
export const UpdateHealthSettingsBody = HealthSettingsInputBody;

export const UpdateHealthSettingsResponse = HealthSettings;

/**
 * @summary Remove a volume's own levels
 */

export const DeleteHealthVolumeQueryParams = zod.object({
  nodeId: zod.uuid(),
  mount: zod.string().min(1),
});

export const DeleteHealthVolumeResponse = zod.void();

/**
 * @summary List every volume with the levels it follows
 */
export const ListHealthVolumesResponse = HealthVolumeList;

/**
 * @summary Set or reset a volume's own levels
 */
export const UpdateHealthVolumeBody = HealthVolumeInputBody;

export const UpdateHealthVolumeResponse = HealthVolume;

/**
 * @summary Check one category now
 */
export const CheckHealthCategoryParams = zod.object({
  category: zod.string(),
});

export const CheckHealthCategoryResponse = HealthChecked;

/**
 * @summary Check one item now
 */
export const CheckHealthItemParams = zod.object({
  category: zod.string(),
  key: zod.string(),
});

export const CheckHealthItemResponse = HealthChecked;

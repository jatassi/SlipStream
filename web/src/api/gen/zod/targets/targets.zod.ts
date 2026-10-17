import * as zod from "zod";

import {
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsProfile,
  TargetsProfileList,
  TargetsQualityList,
  TargetsTarget,
  TargetsTargetList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Profiles
 */
export const ListProfilesResponse = TargetsProfileList;

/**
 * @summary Create a Profile; a Content Type's first is its default
 */
export const CreateProfileBody = TargetsNewProfile;

export const CreateProfileResponse = TargetsProfile;

/**
 * @summary List the qualities a Target can accept, and the presets that order them
 */
export const ListQualitiesResponse = TargetsQualityList;

/**
 * @summary List the Targets
 */
export const ListTargetsResponse = TargetsTargetList;

/**
 * @summary Create a Target from its tiers of qualities, cutoff, filters and sizes
 */
export const CreateTargetBody = TargetsNewTarget;

export const CreateTargetResponse = TargetsTarget;

/**
 * @summary Delete a Target no Profile holds
 */
export const DeleteTargetParams = zod.object({
  targetId: zod.uuid(),
});

export const DeleteTargetResponse = zod.void();

/**
 * @summary Get a Target
 */
export const GetTargetParams = zod.object({
  targetId: zod.uuid(),
});

export const GetTargetResponse = TargetsTarget;

/**
 * @summary Copy a Target's rules under the first free name ending in copy
 */
export const DuplicateTargetParams = zod.object({
  targetId: zod.uuid(),
});

export const DuplicateTargetResponse = TargetsTarget;

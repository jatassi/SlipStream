import {
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsProfile,
  TargetsProfileList,
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
 * @summary List the Targets
 */
export const ListTargetsResponse = TargetsTargetList;

/**
 * @summary Create a Target from its resolutions
 */
export const CreateTargetBody = TargetsNewTarget;

export const CreateTargetResponse = TargetsTarget;

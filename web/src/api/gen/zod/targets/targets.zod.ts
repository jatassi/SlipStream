import * as zod from "zod";

import {
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsProfile,
  TargetsProfileDetail,
  TargetsProfileList,
  TargetsQualityList,
  TargetsTarget,
  TargetsTargetList,
  TargetsWants,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Profiles, each with the count of Works wanting from it
 */
export const ListProfilesResponse = TargetsProfileList;

/**
 * @summary Create a Profile of Targets in Profile order; it is never the default
 */
export const CreateProfileBody = TargetsNewProfile;

export const CreateProfileResponse = TargetsProfile;

/**
 * @summary Get a Profile with the disk each of its Targets costs across its Works
 */
export const GetProfileParams = zod.object({
  profileId: zod.uuid(),
});

export const GetProfileResponse = TargetsProfileDetail;

/**
 * @summary Make a Profile its Content Type's default; existing Works keep theirs
 */
export const MakeDefaultProfileParams = zod.object({
  profileId: zod.uuid(),
});

export const MakeDefaultProfileResponse = TargetsProfile;

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

/**
 * @summary Get the Profile a Work wants from and its Edition list
 */
export const GetWorkWantsParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkWantsResponse = TargetsWants;

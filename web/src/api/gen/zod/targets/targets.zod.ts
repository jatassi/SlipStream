import * as zod from "zod";

import {
  TargetsCut,
  TargetsCutChange,
  TargetsCutList,
  TargetsNewCut,
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsNewWants,
  TargetsProfile,
  TargetsProfileChange,
  TargetsProfileDetail,
  TargetsProfileList,
  TargetsQualityList,
  TargetsTarget,
  TargetsTargetChange,
  TargetsTargetList,
  TargetsWants,
  TargetsWantsChange,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the cuts an Edition list can name: built in, added by a Content Type, or custom
 */
export const ListCutsResponse = TargetsCutList;

/**
 * @summary Add a custom cut, keyed by its name's words, which every title parse reads from then on
 */
export const CreateCutBody = TargetsNewCut;

export const CreateCutResponse = TargetsCut;

/**
 * @summary Remove a custom cut no Edition list names and no file is of
 */
export const DeleteCutParams = zod.object({
  cutKey: zod.string(),
});

export const DeleteCutResponse = zod.void();

/**
 * @summary Rename a custom cut or replace its aliases, renaming the files of it, or preview the change's Impact
 */
export const UpdateCutParams = zod.object({
  cutKey: zod.string(),
});

export const updateCutQueryUnwantedDefault = `delete`;
export const updateCutQueryConfirmMin = 0;

export const updateCutQueryConfirmMadeMin = 0;

export const UpdateCutQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(updateCutQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(updateCutQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(updateCutQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const UpdateCutBody = TargetsNewCut;

export const UpdateCutResponse = TargetsCutChange;

/**
 * @summary List the Profiles, each with the number of Works that use it
 */
export const ListProfilesResponse = TargetsProfileList;

/**
 * @summary Create a Profile of Targets in Profile order. A new Profile is never the default
 */
export const CreateProfileBody = TargetsNewProfile;

export const CreateProfileResponse = TargetsProfile;

/**
 * @summary Delete a Profile that isn't its Content Type's default, moving its Works to another of the type, or preview the change's Impact
 */
export const DeleteProfileParams = zod.object({
  profileId: zod.uuid(),
});

export const deleteProfileQueryUnwantedDefault = `delete`;
export const deleteProfileQueryConfirmMin = 0;

export const deleteProfileQueryConfirmMadeMin = 0;

export const DeleteProfileQueryParams = zod.object({
  moveTo: zod.uuid().describe("The Profile of the same Content Type that its Works move to"),
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(deleteProfileQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(deleteProfileQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(deleteProfileQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const DeleteProfileResponse = TargetsProfileChange;

/**
 * @summary Get a Profile with the disk each of its Targets costs across its Works
 */
export const GetProfileParams = zod.object({
  profileId: zod.uuid(),
});

export const GetProfileResponse = TargetsProfileDetail;

/**
 * @summary Replace a Profile's name, Targets, Edition list and subtitle languages, refilling and renaming its Works' files when its Targets or Edition list change, or preview the change's Impact
 */
export const UpdateProfileParams = zod.object({
  profileId: zod.uuid(),
});

export const updateProfileQueryConfirmSubtitlesMin = 0;

export const updateProfileQueryUnwantedDefault = `delete`;
export const updateProfileQueryConfirmMin = 0;

export const updateProfileQueryConfirmMadeMin = 0;

export const UpdateProfileQueryParams = zod.object({
  confirmSubtitles: zod
    .int()
    .min(updateProfileQueryConfirmSubtitlesMin)
    .exactOptional()
    .describe(
      "The deleted subtitle count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(updateProfileQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(updateProfileQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(updateProfileQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const UpdateProfileBody = TargetsNewProfile;

export const UpdateProfileResponse = TargetsProfileChange;

/**
 * @summary Make a Profile its Content Type's default. Existing Works keep theirs
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
 * @summary Create a Target from its tiers of qualities, cutoff, filters, sizes, Custom Format scores and propers setting
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
 * @summary Replace a Target's name and rules, renaming and refilling its Works' files, or preview the change's Impact
 */
export const UpdateTargetParams = zod.object({
  targetId: zod.uuid(),
});

export const updateTargetQueryConfirmRemakeMin = 0;

export const updateTargetQueryUnwantedDefault = `delete`;
export const updateTargetQueryConfirmMin = 0;

export const updateTargetQueryConfirmMadeMin = 0;

export const UpdateTargetQueryParams = zod.object({
  confirmRemake: zod
    .int()
    .min(updateTargetQueryConfirmRemakeMin)
    .exactOptional()
    .describe(
      "The remade file count the Admin confirmed from the preview: a save that would remake more is refused with impact_changed",
    ),
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(updateTargetQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(updateTargetQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(updateTargetQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const UpdateTargetBody = TargetsNewTarget;

export const UpdateTargetResponse = TargetsTargetChange;

/**
 * @summary Copy a Target's rules under the first free name ending in copy
 */
export const DuplicateTargetParams = zod.object({
  targetId: zod.uuid(),
});

export const DuplicateTargetResponse = TargetsTarget;

/**
 * @summary Get the Profile a Work uses and its Edition list
 */
export const GetWorkWantsParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkWantsResponse = TargetsWants;

/**
 * @summary Set a Work's own Edition list, or its Profile's, refilling and renaming its files, or preview the change's Impact
 */
export const SetWorkWantsParams = zod.object({
  workId: zod.uuid(),
});

export const setWorkWantsQueryUnwantedDefault = `delete`;
export const setWorkWantsQueryConfirmMin = 0;

export const setWorkWantsQueryConfirmMadeMin = 0;

export const SetWorkWantsQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(setWorkWantsQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(setWorkWantsQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(setWorkWantsQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const SetWorkWantsBody = TargetsNewWants;

export const SetWorkWantsResponse = TargetsWantsChange;

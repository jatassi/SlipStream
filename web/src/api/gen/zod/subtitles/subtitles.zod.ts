import * as zod from "zod";

import {
  SubtitlesAttempt,
  SubtitlesAttempting,
  SubtitlesCandidates,
  SubtitlesNodeSettings,
  SubtitlesProvider,
  SubtitlesProviderChange,
  SubtitlesProviderKey,
  SubtitlesProviderList,
  SubtitlesProviderOrder,
  SubtitlesStarted,
  SubtitlesSubtitleSettings,
  SubtitlesUpload,
  SubtitlesValuesBody,
  SubtitlesWorkSubtitles,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get a subtitle download's or upload's step, and its outcome once finished
 */
export const GetSubtitleAttemptParams = zod.object({
  fileId: zod.uuid(),
  attemptId: zod.uuid(),
});

export const GetSubtitleAttemptResponse = SubtitlesAttempt;

/**
 * @summary Count an embedded subtitle track toward the file's wants again
 */
export const countSubtitleTrackPathStreamMin = 0;

export const CountSubtitleTrackParams = zod.object({
  fileId: zod.uuid(),
  stream: zod.int().min(countSubtitleTrackPathStreamMin),
});

export const CountSubtitleTrackResponse = zod.void();

/**
 * @summary Don't count an embedded subtitle track toward the file's wants
 */
export const ignoreSubtitleTrackPathStreamMin = 0;

export const IgnoreSubtitleTrackParams = zod.object({
  fileId: zod.uuid(),
  stream: zod.int().min(ignoreSubtitleTrackPathStreamMin),
});

export const IgnoreSubtitleTrackResponse = zod.void();

/**
 * @summary Keep a refused upload without a timing check
 */
export const KeepSubtitleUploadParams = zod.object({
  fileId: zod.uuid(),
  uploadId: zod.uuid(),
});

export const KeepSubtitleUploadResponse = zod.void();

/**
 * @summary Start a Manual subtitle search on one file's wanted subtitle
 */
export const StartSubtitleCandidatesParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
});

export const StartSubtitleCandidatesResponse = SubtitlesStarted;

/**
 * @summary Get a Manual subtitle search's ranked candidates so far
 */
export const GetSubtitleCandidatesParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
  searchId: zod.uuid(),
});

export const GetSubtitleCandidatesResponse = SubtitlesCandidates;

/**
 * @summary Download a candidate and check its timing against the file
 */
export const PickSubtitleCandidateParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
  searchId: zod.uuid(),
  key: zod.string().describe("The candidate's key, from the search"),
});

export const PickSubtitleCandidateResponse = SubtitlesAttempting;

/**
 * @summary Delete a file's subtitle, never use it for the file again and search for another
 */
export const MarkSubtitleBadParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
});

export const MarkSubtitleBadResponse = zod.void();

/**
 * @summary Search for one file's wanted subtitle now, ahead of the schedule
 */
export const SearchSubtitleParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
});

export const SearchSubtitleResponse = zod.void();

/**
 * @summary Upload a subtitle file and check its timing against the file
 */
export const UploadSubtitleParams = zod.object({
  fileId: zod.uuid(),
  want: zod.string().describe("en, pt-BR or en-forced"),
});

export const UploadSubtitleBody = SubtitlesUpload;

export const UploadSubtitleResponse = SubtitlesAttempting;

/**
 * @summary Get a Storage Node's Subtitles settings
 */
export const GetSubtitleNodeSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const GetSubtitleNodeSettingsResponse = SubtitlesNodeSettings;

/**
 * @summary Change a Storage Node's Subtitles settings by key, all or none
 */
export const UpdateSubtitleNodeSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const UpdateSubtitleNodeSettingsBody = SubtitlesValuesBody;

export const UpdateSubtitleNodeSettingsResponse = SubtitlesNodeSettings;

/**
 * @summary List the Subtitle Providers in search order
 */
export const ListSubtitleProvidersResponse = SubtitlesProviderList;

/**
 * @summary Set the Subtitle Providers' search order
 */
export const OrderSubtitleProvidersBody = SubtitlesProviderOrder;

export const OrderSubtitleProvidersResponse = SubtitlesProviderList;

/**
 * @summary Turn a Subtitle Provider on or off
 */
export const UpdateSubtitleProviderParams = zod.object({
  id: zod.string(),
});

export const UpdateSubtitleProviderBody = SubtitlesProviderChange;

export const UpdateSubtitleProviderResponse = SubtitlesProvider;

/**
 * @summary Forget a Subtitle Provider's key
 */
export const DeleteSubtitleProviderKeyParams = zod.object({
  id: zod.string(),
});

export const DeleteSubtitleProviderKeyResponse = SubtitlesProvider;

/**
 * @summary Check a Subtitle Provider key on a Subtitles holder, then save it
 */
export const SetSubtitleProviderKeyParams = zod.object({
  id: zod.string(),
});

export const SetSubtitleProviderKeyBody = SubtitlesProviderKey;

export const SetSubtitleProviderKeyResponse = SubtitlesProvider;

/**
 * @summary Get the Subtitles settings, the section's lead and the Profiles with Wanted Languages
 */
export const GetSubtitleSettingsResponse = SubtitlesSubtitleSettings;

/**
 * @summary Change Subtitles settings by key, all or none
 */
export const UpdateSubtitleSettingsBody = SubtitlesValuesBody;

export const UpdateSubtitleSettingsResponse = SubtitlesSubtitleSettings;

/**
 * @summary Get each library file of a Work with what each of its wanted subtitles reads as
 */
export const GetWorkSubtitlesParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkSubtitlesResponse = SubtitlesWorkSubtitles;

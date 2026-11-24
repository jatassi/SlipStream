import * as zod from "zod";

import {
  ReachabilityExposurePublic,
  ReachabilityExposureView,
  ReachabilityNewExposure,
  ReachabilityNodeNetwork,
  ReachabilityRemoteAccess,
  ReachabilityRunStarted,
  ReachabilitySettingsBody,
  ReachabilitySettingsValues,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get Remote access: every route set up for use away from home with its verdict, the suggestions and each Node's class
 */
export const GetRemoteAccessResponse = ReachabilityRemoteAccess;

/**
 * @summary Set the UI and portal, or a Media Server, up for use away from home, and check it
 */
export const AddExposureBody = ReachabilityNewExposure;

export const AddExposureResponse = ReachabilityExposureView;

/**
 * @summary Stop checking a route, and forget its result
 */
export const RemoveExposureParams = zod.object({
  id: zod.uuid(),
});

export const RemoveExposureResponse = zod.void();

/**
 * @summary Change a Media Server's public address, and check it again
 */
export const EditExposureParams = zod.object({
  id: zod.uuid(),
});

export const EditExposureBody = ReachabilityExposurePublic;

export const EditExposureResponse = ReachabilityExposureView;

/**
 * @summary Get a Node's Network section, read now with nothing sent outside the machine
 */
export const GetNodeNetworkParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeNetworkResponse = ReachabilityNodeNetwork;

/**
 * @summary Get a Node's Remote access settings
 */
export const GetNodeNetworkSettingsParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeNetworkSettingsResponse = ReachabilitySettingsBody;

/**
 * @summary Change a Node's Remote access settings by key, all or none
 */
export const PatchNodeNetworkSettingsParams = zod.object({
  id: zod.uuid(),
});

export const PatchNodeNetworkSettingsBody = ReachabilitySettingsValues;

export const PatchNodeNetworkSettingsResponse = ReachabilitySettingsBody;

/**
 * @summary Check every route again, or one, with a fresh look at each Node's network
 */
export const RunRemoteAccessQueryParams = zod.object({
  exposure: zod.uuid().exactOptional().describe("Left out, every route"),
});

export const RunRemoteAccessResponse = ReachabilityRunStarted;

/**
 * @summary Get the Remote access settings of Settings > Access
 */
export const GetRemoteAccessSettingsResponse = ReachabilitySettingsBody;

/**
 * @summary Change the Remote access settings by key, all or none
 */
export const PatchRemoteAccessSettingsBody = ReachabilitySettingsValues;

export const PatchRemoteAccessSettingsResponse = ReachabilitySettingsBody;

/**
 * @summary Hide a suggestion: ui-portal, or a Media Server's id
 */
export const DismissSuggestionParams = zod.object({
  id: zod.string().describe("ui-portal, or a Media Server's id"),
});

export const DismissSuggestionResponse = zod.void();

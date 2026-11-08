import * as zod from "zod";

import {
  MergeTrackerInputBody,
  RenameTrackerInputBody,
  TrackersChoice,
  TrackersGoal,
  TrackersGoalsPreview,
  TrackersRouteList,
  TrackersRuleBody,
  TrackersSettings,
  TrackersSettingsChange,
  TrackersTracker,
  TrackersTrackerList,
  TrackersTrackerPage,
  TrackersWindowList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List what the next Seeding goals run would do: the torrents whose goal acts, the ones it removes keeping their download copy, and the ones whose goal is met but waits
 */
export const PreviewSeedingGoalsResponse = TrackersGoalsPreview;

/**
 * @summary List the Trackers and the Indexers that reach each
 */
export const ListTrackersResponse = TrackersTrackerList;

/**
 * @summary Get the seeding goal of every Tracker with none of its own
 */
export const GetDefaultGoalResponse = TrackersGoal;

/**
 * @summary Set the seeding goal of every Tracker with none of its own
 */
export const SetDefaultGoalBody = TrackersGoal;

export const SetDefaultGoalResponse = TrackersGoal;

/**
 * @summary Get the seeding settings
 */
export const GetSeedingSettingsResponse = TrackersSettings;

/**
 * @summary Change seeding settings by key, all or none
 */
export const UpdateSeedingSettingsBody = TrackersSettingsChange;

export const UpdateSeedingSettingsResponse = TrackersSettings;

/**
 * @summary Remove a Tracker that no Indexer reaches and no live torrent or unfinished Grab names. Removing it also deletes its seeding rule, goal and seeding history. Its finished Grabs and removed torrents are kept with no Tracker.
 */
export const RemoveTrackerParams = zod.object({
  id: zod.uuid(),
});

export const RemoveTrackerResponse = zod.void();

/**
 * @summary Get a Tracker
 */
export const GetTrackerParams = zod.object({
  id: zod.uuid(),
});

export const GetTrackerResponse = TrackersTrackerPage;

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const RenameTrackerParams = zod.object({
  id: zod.uuid(),
});

export const RenameTrackerBody = RenameTrackerInputBody;

export const RenameTrackerResponse = TrackersTracker;

/**
 * @summary Delete a Tracker's own seeding goal, so it follows the default goal
 */
export const ClearTrackerGoalParams = zod.object({
  id: zod.uuid(),
});

export const ClearTrackerGoalResponse = TrackersTracker;

/**
 * @summary Set a Tracker's own seeding goal. A goal equal to the default goal is deleted, so the Tracker follows the default.
 */
export const SetTrackerGoalParams = zod.object({
  id: zod.uuid(),
});

export const SetTrackerGoalBody = TrackersGoal;

export const SetTrackerGoalResponse = TrackersTracker;

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents, hosts and seeding history. The merged Tracker is deleted.
 */
export const MergeTrackerParams = zod.object({
  id: zod.uuid(),
});

export const MergeTrackerBody = MergeTrackerInputBody;

export const MergeTrackerResponse = TrackersTracker;

/**
 * @summary List where a private Tracker's Grabs download from the home Node of each of its Indexers
 */
export const ListTrackerRoutesParams = zod.object({
  id: zod.uuid(),
});

export const ListTrackerRoutesResponse = TrackersRouteList;

/**
 * @summary Delete the client picked for a private Tracker's Grabs from Indexers on a home Node. They then download on its built-in client while it holds Downloading, and where the Work's library root is while it doesn't.
 */
export const ClearTrackerRouteParams = zod.object({
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The home Node"),
});

export const ClearTrackerRouteResponse = TrackersRouteList;

/**
 * @summary Send a private Tracker's Grabs from Indexers on a home Node to a client you pick. They download there while it is on and its Node holds Downloading.
 */
export const SetTrackerRouteParams = zod.object({
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The home Node"),
});

export const SetTrackerRouteBody = TrackersChoice;

export const SetTrackerRouteResponse = TrackersRouteList;

/**
 * @summary Delete the seeding rule an Admin set for a Tracker, so the latest rule its Releases state is in force
 */
export const ClearTrackerRuleParams = zod.object({
  id: zod.uuid(),
});

export const ClearTrackerRuleResponse = TrackersTracker;

/**
 * @summary Set a Tracker's seeding rule. Rules its Releases state never replace it.
 */
export const SetTrackerRuleParams = zod.object({
  id: zod.uuid(),
});

export const SetTrackerRuleBody = TrackersRuleBody;

export const SetTrackerRuleResponse = TrackersTracker;

/**
 * @summary List a Tracker's hit and run windows
 */
export const ListTrackerWindowsParams = zod.object({
  id: zod.uuid(),
});

export const listTrackerWindowsQueryOffsetMin = 0;

export const ListTrackerWindowsQueryParams = zod.object({
  state: zod
    .enum(["owing", "met", "missed", "released"])
    .exactOptional()
    .describe("Only windows in this state; every state when absent"),
  offset: zod
    .int()
    .min(listTrackerWindowsQueryOffsetMin)
    .exactOptional()
    .describe("How many of the newest to skip"),
});

export const ListTrackerWindowsResponse = TrackersWindowList;

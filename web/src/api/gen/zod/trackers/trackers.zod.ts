import * as zod from "zod";

import {
  MergeTrackerInputBody,
  RenameTrackerInputBody,
  TrackersGoal,
  TrackersRuleBody,
  TrackersSettings,
  TrackersSettingsChange,
  TrackersTracker,
  TrackersTrackerList,
  TrackersTrackerPage,
} from "../slipStream.schemas.zod.ts";

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

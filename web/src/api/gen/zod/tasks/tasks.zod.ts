import * as zod from "zod";

import {
  TasksDetail,
  TasksOpenRuns,
  TasksOverview,
  TasksRunStarted,
  TasksSchedule,
  TasksSchedulePatch,
  TasksTask,
  TasksTaskPatch,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List every shown task by group, with the status sentence and the schedule settings
 */
export const ListTasksResponse = TasksOverview;

/**
 * @summary List every open run, with how long a scheduled one waits before the header shows it
 */
export const ListOpenTaskRunsResponse = TasksOpenRuns;

/**
 * @summary Cancel an open run
 */
export const CancelTaskRunParams = zod.object({
  runId: zod.int(),
});

export const CancelTaskRunResponse = zod.void();

/**
 * @summary Get the schedule settings
 */
export const GetTaskSettingsResponse = TasksSchedule;

/**
 * @summary Change the schedule settings
 */
export const UpdateTaskSettingsBody = TasksSchedulePatch;

export const UpdateTaskSettingsResponse = TasksSchedule;

/**
 * @summary Get one shown task with its units and its runs
 */
export const GetTaskParams = zod.object({
  taskId: zod.string(),
});

export const GetTaskResponse = TasksDetail;

/**
 * @summary Change a task's cadence or overdue rule
 */
export const UpdateTaskParams = zod.object({
  taskId: zod.string(),
});

export const UpdateTaskBody = TasksTaskPatch;

export const UpdateTaskResponse = TasksTask;

/**
 * @summary Run a task now over every unit; its schedule stays as it was
 */
export const RunTaskParams = zod.object({
  taskId: zod.string(),
});

export const RunTaskResponse = TasksRunStarted;

import type { TasksIdleState } from "./tasksIdleState.ts";
import type { TasksRunningState } from "./tasksRunningState.ts";
import type { TasksWaitingState } from "./tasksWaitingState.ts";

export type TasksState = TasksIdleState | TasksRunningState | TasksWaitingState;

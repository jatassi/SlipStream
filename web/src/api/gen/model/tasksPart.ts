import type { TasksFailurePart } from "./tasksFailurePart.ts";
import type { TasksPlainPart } from "./tasksPlainPart.ts";
import type { TasksWarningPart } from "./tasksWarningPart.ts";

export type TasksPart = TasksPlainPart | TasksWarningPart | TasksFailurePart;

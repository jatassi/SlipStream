import type { TasksUnitStep } from "./tasksUnitStep.ts";

export interface TasksProgressed {
  done: number;
  runId: number;
  taskId: string;
  total: number;
  /** The unit whose step moved; absent when a unit ended */
  unit?: TasksUnitStep;
}

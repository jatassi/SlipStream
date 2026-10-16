import type { TasksOpenRun } from "./tasksOpenRun.ts";

export interface TasksOpenRuns {
  /** How long a scheduled run is open before the header shows it; a run-now run shows at once */
  headerDelaySeconds: number;
  /** Oldest first */
  runs: TasksOpenRun[];
}

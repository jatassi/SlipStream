import type { FilesWaitingNeed } from "./filesWaitingNeed.ts";
import type { GrabsGrab } from "./grabsGrab.ts";
import type { HomeWork } from "./homeWork.ts";

export interface HomeNeed {
  grab: GrabsGrab;
  /** What it waits on, as the Work page words it; null for a failed Grab */
  wait: FilesWaitingNeed | null;
  work: HomeWork;
}

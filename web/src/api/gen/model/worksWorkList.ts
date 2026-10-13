import type { WorksListedWork } from "./worksListedWork.ts";

export interface WorksWorkList {
  /** By sort label, then id */
  works: WorksListedWork[];
}

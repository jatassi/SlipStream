import type { ReachabilityNodeName } from "./reachabilityNodeName.ts";

export interface ReachabilityLastRun {
  at: string;
  testers: ReachabilityNodeName[];
}

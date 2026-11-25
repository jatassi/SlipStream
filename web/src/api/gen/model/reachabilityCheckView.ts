import type { ReachabilityCheckID } from "./reachabilityCheckID.ts";
import type { ReachabilityCheckViewFamily } from "./reachabilityCheckViewFamily.ts";
import type { ReachabilityCheckViewPoints } from "./reachabilityCheckViewPoints.ts";
import type { ReachabilityCheckViewResult } from "./reachabilityCheckViewResult.ts";
import type { ReachabilityFixView } from "./reachabilityFixView.ts";

export interface ReachabilityCheckView {
  /** The Node that saw it */
  by?: string;
  /** Shown, never pointing: an overruled local guess or a connect-back */
  fact: boolean;
  family?: ReachabilityCheckViewFamily;
  fix: ReachabilityFixView | null;
  /** What a fact check that doesn't pass found */
  found?: string;
  id: ReachabilityCheckID;
  meaning: string;
  observed?: string;
  /** What a failing or untested result points to */
  points?: ReachabilityCheckViewPoints;
  result: ReachabilityCheckViewResult;
}

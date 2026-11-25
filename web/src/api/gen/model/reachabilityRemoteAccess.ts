import type { ReachabilityCard } from "./reachabilityCard.ts";
import type { ReachabilityClassLine } from "./reachabilityClassLine.ts";
import type { ReachabilityLastRun } from "./reachabilityLastRun.ts";
import type { ReachabilityRoute } from "./reachabilityRoute.ts";
import type { ReachabilityStatus } from "./reachabilityStatus.ts";
import type { ReachabilitySuggestionView } from "./reachabilitySuggestionView.ts";

export interface ReachabilityRemoteAccess {
  /** The one recommendation, null when every route works and was tested */
  card: ReachabilityCard | null;
  /** Null before the first check */
  lastRun: ReachabilityLastRun | null;
  /** One line per Node, the Controller first; empty while nothing is set up */
  nodes: ReachabilityClassLine[];
  /** Nothing is set up for use away from home: the page shows its Empty state */
  nothingWanted: boolean;
  routes: ReachabilityRoute[];
  status: ReachabilityStatus;
  suggestions: ReachabilitySuggestionView[];
}

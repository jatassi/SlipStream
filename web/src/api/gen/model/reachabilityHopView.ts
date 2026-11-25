import type { ReachabilityCheckView } from "./reachabilityCheckView.ts";
import type { ReachabilityFactView } from "./reachabilityFactView.ts";
import type { ReachabilityHopViewKind } from "./reachabilityHopViewKind.ts";
import type { ReachabilityHopViewWorst } from "./reachabilityHopViewWorst.ts";

export interface ReachabilityHopView {
  checks: ReachabilityCheckView[];
  facts: ReachabilityFactView[];
  kind: ReachabilityHopViewKind;
  worst: ReachabilityHopViewWorst;
}

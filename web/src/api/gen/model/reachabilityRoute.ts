import type { ReachabilityExposureView } from "./reachabilityExposureView.ts";
import type { ReachabilityFactView } from "./reachabilityFactView.ts";
import type { ReachabilityFixView } from "./reachabilityFixView.ts";
import type { ReachabilityHopView } from "./reachabilityHopView.ts";
import type { ReachabilityRouteCard } from "./reachabilityRouteCard.ts";
import type { ReachabilityRouteRoute } from "./reachabilityRouteRoute.ts";
import type { ReachabilityRouteVerdict } from "./reachabilityRouteVerdict.ts";

export interface ReachabilityRoute {
  card: ReachabilityRouteCard;
  /** False until the first check of this route finished */
  checked: boolean;
  exposure: ReachabilityExposureView;
  facts: ReachabilityFactView[];
  /** The families the route offers, as its tag */
  families?: string;
  /** The step that fixes the deciding check, null with none */
  fix: ReachabilityFixView | null;
  /** Reachable and tested: the route shows as one line */
  folded: boolean;
  hops: ReachabilityHopView[];
  /** @nullable */
  judgedAt: string | null;
  route?: ReachabilityRouteRoute;
  /**
   * Why the route wasn't checked since its last result, null while its origin is in step
   * @nullable
   */
  skipped: string | null;
  /** Every offered family was tested from outside */
  tested: boolean;
  testedBy?: string;
  verdict?: ReachabilityRouteVerdict;
}

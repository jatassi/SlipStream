import type { ReachabilityGatewayView } from "./reachabilityGatewayView.ts";
import type { ReachabilityIfaceView } from "./reachabilityIfaceView.ts";
import type { ReachabilityListenView } from "./reachabilityListenView.ts";
import type { ReachabilityMappingView } from "./reachabilityMappingView.ts";
import type { ReachabilityRuleView } from "./reachabilityRuleView.ts";

export interface ReachabilityNodeNetwork {
  firewall: ReachabilityRuleView[];
  /** Null when no router was asked */
  gateway: ReachabilityGatewayView | null;
  interfaces: ReachabilityIfaceView[];
  /** The Node's class, in no signal color */
  line: string;
  listeners: ReachabilityListenView[];
  mappings: ReachabilityMappingView[];
  name: string;
  nodeId: string;
  /** Read on opening; false is the last stored look, for a Node that is away */
  now: boolean;
  /** @nullable */
  observedAt: string | null;
}

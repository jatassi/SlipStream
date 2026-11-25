import type { ReachabilityRuleViewFamily } from "./reachabilityRuleViewFamily.ts";
import type { ReachabilityRuleViewPolicy } from "./reachabilityRuleViewPolicy.ts";

export interface ReachabilityRuleView {
  family: ReachabilityRuleViewFamily;
  policy: ReachabilityRuleViewPolicy;
  port: number;
}

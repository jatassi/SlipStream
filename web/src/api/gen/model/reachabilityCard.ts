import type { ReachabilityCardKind } from "./reachabilityCardKind.ts";
import type { ReachabilityCardVerdict } from "./reachabilityCardVerdict.ts";
import type { ReachabilityFixView } from "./reachabilityFixView.ts";

export interface ReachabilityCard {
  exposureId: string;
  fix: ReachabilityFixView;
  kind: ReachabilityCardKind;
  subject: string;
  verdict: ReachabilityCardVerdict;
}

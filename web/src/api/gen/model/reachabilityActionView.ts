import type { ReachabilityActionViewKind } from "./reachabilityActionViewKind.ts";

export interface ReachabilityActionView {
  kind: ReachabilityActionViewKind;
  label: string;
  nodeId?: string;
}

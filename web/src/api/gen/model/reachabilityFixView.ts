import type { ReachabilityActionView } from "./reachabilityActionView.ts";
import type { ReachabilityFixID } from "./reachabilityFixID.ts";

export interface ReachabilityFixView {
  /** The fix's button, null with none */
  action: ReachabilityActionView;
  /** The firewall command, when the fix carries one */
  command?: string;
  details?: string;
  id: ReachabilityFixID;
  step: string;
  title: string;
}

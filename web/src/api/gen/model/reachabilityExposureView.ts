import type { ReachabilityExposureViewKind } from "./reachabilityExposureViewKind.ts";

export interface ReachabilityExposureView {
  id: string;
  kind: ReachabilityExposureViewKind;
  /** The address players outside the home use; the UI and portal's is Address */
  public: string;
  /** SlipStream UI and portal, or the Media Server's name */
  subject: string;
}

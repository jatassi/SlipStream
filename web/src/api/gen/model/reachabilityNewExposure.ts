import type { ReachabilityNewExposureKind } from "./reachabilityNewExposureKind.ts";

export interface ReachabilityNewExposure {
  kind: ReachabilityNewExposureKind;
  /** The address players outside the home use, for kind media-server */
  public?: string;
  /** The Media Server, for kind media-server */
  serverId?: string;
}

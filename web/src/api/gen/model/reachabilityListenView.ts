import type { ReachabilityListenViewIpv4 } from "./reachabilityListenViewIpv4.ts";
import type { ReachabilityListenViewIpv6 } from "./reachabilityListenViewIpv6.ts";

export interface ReachabilityListenView {
  ipv4: ReachabilityListenViewIpv4;
  ipv6: ReachabilityListenViewIpv6;
  loopback: boolean;
  port: number;
  /** Empty when SlipStream couldn't see the owner */
  program: string;
}

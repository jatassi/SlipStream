import type { ReachabilityStatusPartTone } from "./reachabilityStatusPartTone.ts";

export interface ReachabilityStatusPart {
  /** The route the part is about, which it links to */
  exposureId?: string;
  text: string;
  tone: ReachabilityStatusPartTone;
}

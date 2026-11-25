import type { OfferMediaServer502Code } from "./offerMediaServer502Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer502 = Problem & {
  code?: OfferMediaServer502Code;
};

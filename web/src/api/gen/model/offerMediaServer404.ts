import type { OfferMediaServer404Code } from "./offerMediaServer404Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer404 = Problem & {
  code?: OfferMediaServer404Code;
};

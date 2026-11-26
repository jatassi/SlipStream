import type { OfferMediaServer422Code } from "./offerMediaServer422Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer422 = Problem & {
  code?: OfferMediaServer422Code;
};

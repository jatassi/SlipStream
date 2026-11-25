import type { OfferMediaServer401Code } from "./offerMediaServer401Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer401 = Problem & {
  code?: OfferMediaServer401Code;
};

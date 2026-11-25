import type { OfferMediaServer500Code } from "./offerMediaServer500Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer500 = Problem & {
  code?: OfferMediaServer500Code;
};

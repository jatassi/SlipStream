import type { OfferMediaServer409Code } from "./offerMediaServer409Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer409 = Problem & {
  code?: OfferMediaServer409Code;
};

import type { OfferMediaServer403Code } from "./offerMediaServer403Code.ts";
import type { Problem } from "./problem.ts";

export type OfferMediaServer403 = Problem & {
  code?: OfferMediaServer403Code;
};

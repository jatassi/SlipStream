import type { Bare } from "./bare.ts";

export interface FormatsKindValue {
  /** As the value field holds it: a number in the radarr and sonarr dialects, a name in slipstream's */
  value: Bare;
  words: string;
}

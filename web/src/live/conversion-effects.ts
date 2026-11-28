import type { LiveEffects } from "./types";

export const conversionEffects = {
  "conversions.changed": () => {},
  "conversions.live": () => {},
} satisfies Pick<LiveEffects, "conversions.changed" | "conversions.live">;

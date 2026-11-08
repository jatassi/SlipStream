import type { ParseReleases400Code } from "./parseReleases400Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases400 = Problem & {
  code?: ParseReleases400Code;
};

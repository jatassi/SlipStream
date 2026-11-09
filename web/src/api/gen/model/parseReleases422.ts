import type { ParseReleases422Code } from "./parseReleases422Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases422 = Problem & {
  code?: ParseReleases422Code;
};

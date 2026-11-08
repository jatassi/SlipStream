import type { ParseReleases401Code } from "./parseReleases401Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases401 = Problem & {
  code?: ParseReleases401Code;
};

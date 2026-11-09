import type { ParseReleases500Code } from "./parseReleases500Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases500 = Problem & {
  code?: ParseReleases500Code;
};

import type { ParseReleases409Code } from "./parseReleases409Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases409 = Problem & {
  code?: ParseReleases409Code;
};

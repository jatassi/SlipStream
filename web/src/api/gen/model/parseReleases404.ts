import type { ParseReleases404Code } from "./parseReleases404Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases404 = Problem & {
  code?: ParseReleases404Code;
};

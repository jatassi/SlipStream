import type { ParseReleases403Code } from "./parseReleases403Code.ts";
import type { Problem } from "./problem.ts";

export type ParseReleases403 = Problem & {
  code?: ParseReleases403Code;
};

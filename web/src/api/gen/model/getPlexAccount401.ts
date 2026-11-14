import type { GetPlexAccount401Code } from "./getPlexAccount401Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexAccount401 = Problem & {
  code?: GetPlexAccount401Code;
};

import type { GetPlexAccount500Code } from "./getPlexAccount500Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexAccount500 = Problem & {
  code?: GetPlexAccount500Code;
};

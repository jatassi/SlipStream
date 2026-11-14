import type { GetPlexAccount403Code } from "./getPlexAccount403Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexAccount403 = Problem & {
  code?: GetPlexAccount403Code;
};

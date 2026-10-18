import type { GetProfile401Code } from "./getProfile401Code.ts";
import type { Problem } from "./problem.ts";

export type GetProfile401 = Problem & {
  code?: GetProfile401Code;
};

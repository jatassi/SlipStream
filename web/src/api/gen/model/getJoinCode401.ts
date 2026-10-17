import type { GetJoinCode401Code } from "./getJoinCode401Code.ts";
import type { Problem } from "./problem.ts";

export type GetJoinCode401 = Problem & {
  code?: GetJoinCode401Code;
};

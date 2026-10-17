import type { GetJoinCode500Code } from "./getJoinCode500Code.ts";
import type { Problem } from "./problem.ts";

export type GetJoinCode500 = Problem & {
  code?: GetJoinCode500Code;
};

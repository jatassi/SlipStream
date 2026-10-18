import type { GetTopology401Code } from "./getTopology401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTopology401 = Problem & {
  code?: GetTopology401Code;
};

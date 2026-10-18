import type { GetTopology500Code } from "./getTopology500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTopology500 = Problem & {
  code?: GetTopology500Code;
};

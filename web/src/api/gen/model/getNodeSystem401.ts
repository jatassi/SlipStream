import type { GetNodeSystem401Code } from "./getNodeSystem401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeSystem401 = Problem & {
  code?: GetNodeSystem401Code;
};

import type { GetNodeSystem500Code } from "./getNodeSystem500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeSystem500 = Problem & {
  code?: GetNodeSystem500Code;
};

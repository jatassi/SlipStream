import type { GetNodeConverting404Code } from "./getNodeConverting404Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConverting404 = Problem & {
  code?: GetNodeConverting404Code;
};

import type { GetNodeSystem404Code } from "./getNodeSystem404Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeSystem404 = Problem & {
  code?: GetNodeSystem404Code;
};

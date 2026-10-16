import type { GetNodeConfig500Code } from "./getNodeConfig500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConfig500 = Problem & {
  code?: GetNodeConfig500Code;
};

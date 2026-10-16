import type { GetNodeConfig404Code } from "./getNodeConfig404Code.ts";
import type { Problem } from "./problem.ts";

export type GetNodeConfig404 = Problem & {
  code?: GetNodeConfig404Code;
};

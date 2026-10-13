import type { Problem } from "./problem.ts";
import type { SetNodeRoles401Code } from "./setNodeRoles401Code.ts";

export type SetNodeRoles401 = Problem & {
  code?: SetNodeRoles401Code;
};

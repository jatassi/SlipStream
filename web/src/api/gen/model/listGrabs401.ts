import type { ListGrabs401Code } from "./listGrabs401Code.ts";
import type { Problem } from "./problem.ts";

export type ListGrabs401 = Problem & {
  code?: ListGrabs401Code;
};

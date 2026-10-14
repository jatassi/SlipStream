import type { ListGrabs500Code } from "./listGrabs500Code.ts";
import type { Problem } from "./problem.ts";

export type ListGrabs500 = Problem & {
  code?: ListGrabs500Code;
};

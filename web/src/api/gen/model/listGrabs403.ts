import type { ListGrabs403Code } from "./listGrabs403Code.ts";
import type { Problem } from "./problem.ts";

export type ListGrabs403 = Problem & {
  code?: ListGrabs403Code;
};

import type { ListPlexServers500Code } from "./listPlexServers500Code.ts";
import type { Problem } from "./problem.ts";

export type ListPlexServers500 = Problem & {
  code?: ListPlexServers500Code;
};

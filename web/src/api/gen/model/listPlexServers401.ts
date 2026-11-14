import type { ListPlexServers401Code } from "./listPlexServers401Code.ts";
import type { Problem } from "./problem.ts";

export type ListPlexServers401 = Problem & {
  code?: ListPlexServers401Code;
};

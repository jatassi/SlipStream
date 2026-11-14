import type { ListMediaServers401Code } from "./listMediaServers401Code.ts";
import type { Problem } from "./problem.ts";

export type ListMediaServers401 = Problem & {
  code?: ListMediaServers401Code;
};

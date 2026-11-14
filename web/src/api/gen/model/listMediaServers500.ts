import type { ListMediaServers500Code } from "./listMediaServers500Code.ts";
import type { Problem } from "./problem.ts";

export type ListMediaServers500 = Problem & {
  code?: ListMediaServers500Code;
};

import type { ListPlexServers403Code } from "./listPlexServers403Code.ts";
import type { Problem } from "./problem.ts";

export type ListPlexServers403 = Problem & {
  code?: ListPlexServers403Code;
};

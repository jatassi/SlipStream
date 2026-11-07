import type { ListTrackers401Code } from "./listTrackers401Code.ts";
import type { Problem } from "./problem.ts";

export type ListTrackers401 = Problem & {
  code?: ListTrackers401Code;
};

import type { ListSessions500Code } from "./listSessions500Code.ts";
import type { Problem } from "./problem.ts";

export type ListSessions500 = Problem & {
  code?: ListSessions500Code;
};

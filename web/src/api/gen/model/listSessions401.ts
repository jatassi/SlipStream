import type { ListSessions401Code } from "./listSessions401Code.ts";
import type { Problem } from "./problem.ts";

export type ListSessions401 = Problem & {
  code?: ListSessions401Code;
};

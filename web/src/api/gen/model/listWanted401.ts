import type { ListWanted401Code } from "./listWanted401Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted401 = Problem & {
  code?: ListWanted401Code;
};

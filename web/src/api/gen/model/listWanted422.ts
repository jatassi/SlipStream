import type { ListWanted422Code } from "./listWanted422Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted422 = Problem & {
  code?: ListWanted422Code;
};

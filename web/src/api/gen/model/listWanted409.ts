import type { ListWanted409Code } from "./listWanted409Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted409 = Problem & {
  code?: ListWanted409Code;
};

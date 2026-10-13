import type { ListWorks422Code } from "./listWorks422Code.ts";
import type { Problem } from "./problem.ts";

export type ListWorks422 = Problem & {
  code?: ListWorks422Code;
};

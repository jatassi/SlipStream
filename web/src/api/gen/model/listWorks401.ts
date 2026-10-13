import type { ListWorks401Code } from "./listWorks401Code.ts";
import type { Problem } from "./problem.ts";

export type ListWorks401 = Problem & {
  code?: ListWorks401Code;
};

import type { ListWorks500Code } from "./listWorks500Code.ts";
import type { Problem } from "./problem.ts";

export type ListWorks500 = Problem & {
  code?: ListWorks500Code;
};

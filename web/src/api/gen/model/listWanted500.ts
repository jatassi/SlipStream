import type { ListWanted500Code } from "./listWanted500Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted500 = Problem & {
  code?: ListWanted500Code;
};

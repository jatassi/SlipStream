import type { ListWanted400Code } from "./listWanted400Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted400 = Problem & {
  code?: ListWanted400Code;
};

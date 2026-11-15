import type { ListWanted501Code } from "./listWanted501Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted501 = Problem & {
  code?: ListWanted501Code;
};

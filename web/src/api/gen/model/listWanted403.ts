import type { ListWanted403Code } from "./listWanted403Code.ts";
import type { Problem } from "./problem.ts";

export type ListWanted403 = Problem & {
  code?: ListWanted403Code;
};

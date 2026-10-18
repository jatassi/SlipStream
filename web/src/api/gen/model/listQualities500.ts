import type { ListQualities500Code } from "./listQualities500Code.ts";
import type { Problem } from "./problem.ts";

export type ListQualities500 = Problem & {
  code?: ListQualities500Code;
};

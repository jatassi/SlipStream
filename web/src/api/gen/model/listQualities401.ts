import type { ListQualities401Code } from "./listQualities401Code.ts";
import type { Problem } from "./problem.ts";

export type ListQualities401 = Problem & {
  code?: ListQualities401Code;
};

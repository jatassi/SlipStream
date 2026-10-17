import type { Problem } from "./problem.ts";
import type { SetNodeDial404Code } from "./setNodeDial404Code.ts";

export type SetNodeDial404 = Problem & {
  code?: SetNodeDial404Code;
};

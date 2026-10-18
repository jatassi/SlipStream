import type { Problem } from "./problem.ts";
import type { SetNodeDial401Code } from "./setNodeDial401Code.ts";

export type SetNodeDial401 = Problem & {
  code?: SetNodeDial401Code;
};

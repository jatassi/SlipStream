import type { Problem } from "./problem.ts";
import type { SetNodeDial500Code } from "./setNodeDial500Code.ts";

export type SetNodeDial500 = Problem & {
  code?: SetNodeDial500Code;
};

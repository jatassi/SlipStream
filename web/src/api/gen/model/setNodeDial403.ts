import type { Problem } from "./problem.ts";
import type { SetNodeDial403Code } from "./setNodeDial403Code.ts";

export type SetNodeDial403 = Problem & {
  code?: SetNodeDial403Code;
};

import type { AddWork500Code } from "./addWork500Code.ts";
import type { Problem } from "./problem.ts";

export type AddWork500 = Problem & {
  code?: AddWork500Code;
};

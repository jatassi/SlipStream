import type { AddWork401Code } from "./addWork401Code.ts";
import type { Problem } from "./problem.ts";

export type AddWork401 = Problem & {
  code?: AddWork401Code;
};

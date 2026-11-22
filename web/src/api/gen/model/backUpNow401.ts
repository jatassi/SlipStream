import type { BackUpNow401Code } from "./backUpNow401Code.ts";
import type { Problem } from "./problem.ts";

export type BackUpNow401 = Problem & {
  code?: BackUpNow401Code;
};

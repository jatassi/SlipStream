import type { BackUpNow409Code } from "./backUpNow409Code.ts";
import type { Problem } from "./problem.ts";

export type BackUpNow409 = Problem & {
  code?: BackUpNow409Code;
};

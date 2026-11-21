import type { BackUpNow500Code } from "./backUpNow500Code.ts";
import type { Problem } from "./problem.ts";

export type BackUpNow500 = Problem & {
  code?: BackUpNow500Code;
};

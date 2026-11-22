import type { BackUpNow403Code } from "./backUpNow403Code.ts";
import type { Problem } from "./problem.ts";

export type BackUpNow403 = Problem & {
  code?: BackUpNow403Code;
};

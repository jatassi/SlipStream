import type { ForgetFiles404Code } from "./forgetFiles404Code.ts";
import type { Problem } from "./problem.ts";

export type ForgetFiles404 = Problem & {
  code?: ForgetFiles404Code;
};

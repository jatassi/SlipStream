import type { ForgetFiles500Code } from "./forgetFiles500Code.ts";
import type { Problem } from "./problem.ts";

export type ForgetFiles500 = Problem & {
  code?: ForgetFiles500Code;
};

import type { ForgetFiles401Code } from "./forgetFiles401Code.ts";
import type { Problem } from "./problem.ts";

export type ForgetFiles401 = Problem & {
  code?: ForgetFiles401Code;
};

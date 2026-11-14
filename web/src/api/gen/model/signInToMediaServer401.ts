import type { Problem } from "./problem.ts";
import type { SignInToMediaServer401Code } from "./signInToMediaServer401Code.ts";

export type SignInToMediaServer401 = Problem & {
  code?: SignInToMediaServer401Code;
};

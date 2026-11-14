import type { Problem } from "./problem.ts";
import type { SignInToMediaServer502Code } from "./signInToMediaServer502Code.ts";

export type SignInToMediaServer502 = Problem & {
  code?: SignInToMediaServer502Code;
};

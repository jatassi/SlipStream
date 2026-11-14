import type { Problem } from "./problem.ts";
import type { SignInToMediaServer404Code } from "./signInToMediaServer404Code.ts";

export type SignInToMediaServer404 = Problem & {
  code?: SignInToMediaServer404Code;
};

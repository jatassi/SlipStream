import type { Problem } from "./problem.ts";
import type { SignInToMediaServer422Code } from "./signInToMediaServer422Code.ts";

export type SignInToMediaServer422 = Problem & {
  code?: SignInToMediaServer422Code;
};

import type { Problem } from "./problem.ts";
import type { SignInToMediaServer501Code } from "./signInToMediaServer501Code.ts";

export type SignInToMediaServer501 = Problem & {
  code?: SignInToMediaServer501Code;
};

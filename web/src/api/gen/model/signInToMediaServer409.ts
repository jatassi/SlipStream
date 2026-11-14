import type { Problem } from "./problem.ts";
import type { SignInToMediaServer409Code } from "./signInToMediaServer409Code.ts";

export type SignInToMediaServer409 = Problem & {
  code?: SignInToMediaServer409Code;
};

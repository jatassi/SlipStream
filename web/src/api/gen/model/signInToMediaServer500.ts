import type { Problem } from "./problem.ts";
import type { SignInToMediaServer500Code } from "./signInToMediaServer500Code.ts";

export type SignInToMediaServer500 = Problem & {
  code?: SignInToMediaServer500Code;
};

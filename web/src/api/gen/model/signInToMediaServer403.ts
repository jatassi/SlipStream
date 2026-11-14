import type { Problem } from "./problem.ts";
import type { SignInToMediaServer403Code } from "./signInToMediaServer403Code.ts";

export type SignInToMediaServer403 = Problem & {
  code?: SignInToMediaServer403Code;
};

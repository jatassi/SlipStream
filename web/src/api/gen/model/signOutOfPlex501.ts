import type { Problem } from "./problem.ts";
import type { SignOutOfPlex501Code } from "./signOutOfPlex501Code.ts";

export type SignOutOfPlex501 = Problem & {
  code?: SignOutOfPlex501Code;
};

import type { Problem } from "./problem.ts";
import type { SignOutOfPlex500Code } from "./signOutOfPlex500Code.ts";

export type SignOutOfPlex500 = Problem & {
  code?: SignOutOfPlex500Code;
};

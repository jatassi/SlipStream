import type { Problem } from "./problem.ts";
import type { SignOutOfPlex401Code } from "./signOutOfPlex401Code.ts";

export type SignOutOfPlex401 = Problem & {
  code?: SignOutOfPlex401Code;
};

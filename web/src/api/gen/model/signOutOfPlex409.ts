import type { Problem } from "./problem.ts";
import type { SignOutOfPlex409Code } from "./signOutOfPlex409Code.ts";

export type SignOutOfPlex409 = Problem & {
  code?: SignOutOfPlex409Code;
};

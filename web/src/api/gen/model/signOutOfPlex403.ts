import type { Problem } from "./problem.ts";
import type { SignOutOfPlex403Code } from "./signOutOfPlex403Code.ts";

export type SignOutOfPlex403 = Problem & {
  code?: SignOutOfPlex403Code;
};

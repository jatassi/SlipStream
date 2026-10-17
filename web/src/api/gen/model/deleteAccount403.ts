import type { DeleteAccount403Code } from "./deleteAccount403Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteAccount403 = Problem & {
  code?: DeleteAccount403Code;
};

import type { DeleteAccount500Code } from "./deleteAccount500Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteAccount500 = Problem & {
  code?: DeleteAccount500Code;
};

import type { GetNode500Code } from "./getNode500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNode500 = Problem & {
  code?: GetNode500Code;
};

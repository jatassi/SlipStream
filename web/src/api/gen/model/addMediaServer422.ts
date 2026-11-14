import type { AddMediaServer422Code } from "./addMediaServer422Code.ts";
import type { Problem } from "./problem.ts";

export type AddMediaServer422 = Problem & {
  code?: AddMediaServer422Code;
};

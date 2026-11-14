import type { AddMediaServer404Code } from "./addMediaServer404Code.ts";
import type { Problem } from "./problem.ts";

export type AddMediaServer404 = Problem & {
  code?: AddMediaServer404Code;
};

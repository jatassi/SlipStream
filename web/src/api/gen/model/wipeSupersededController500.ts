import type { Problem } from "./problem.ts";
import type { WipeSupersededController500Code } from "./wipeSupersededController500Code.ts";

export type WipeSupersededController500 = Problem & {
  code?: WipeSupersededController500Code;
};

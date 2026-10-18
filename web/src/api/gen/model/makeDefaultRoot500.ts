import type { MakeDefaultRoot500Code } from "./makeDefaultRoot500Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultRoot500 = Problem & {
  code?: MakeDefaultRoot500Code;
};

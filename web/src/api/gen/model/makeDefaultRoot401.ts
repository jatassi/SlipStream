import type { MakeDefaultRoot401Code } from "./makeDefaultRoot401Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultRoot401 = Problem & {
  code?: MakeDefaultRoot401Code;
};

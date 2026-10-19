import type { MakeDefaultRoot404Code } from "./makeDefaultRoot404Code.ts";
import type { Problem } from "./problem.ts";

export type MakeDefaultRoot404 = Problem & {
  code?: MakeDefaultRoot404Code;
};
